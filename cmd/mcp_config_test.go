package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

// TestValidateMCPConfig guards the startup refusal that MCP's entire audience
// model rests on.
//
// A token is accepted at /mcp only if its `aud` equals this deployment's
// canonical <url>/mcp. Derive that identifier from request headers — which is
// what parsers.GetHost does when --url is unset, falling back to
// X-Authorizer-URL, X-Forwarded-Host, then Host — and the caller supplies both
// sides of the comparison. Refusing to start is what makes the check mean
// something.
//
// The unusable-URL cases matter as much as the empty one: MCPResource() returns
// "" for a scheme-less or non-http --url too, so starting anyway would produce a
// surface that is enabled, advertises nothing (the metadata handler 404s) and
// rejects every token — broken in the direction that reports success.
func TestValidateMCPConfig(t *testing.T) {
	t.Run("MCP disabled never fails, whatever --url says", func(t *testing.T) {
		for _, u := range []string{"", "not-a-url", "https://auth.example.com"} {
			require.NoError(t, validateMCPConfig(&config.Config{MCPEnabled: false, AuthorizerURL: u}),
				"the guard must not affect deployments that do not run MCP")
		}
	})

	t.Run("MCP enabled with a usable --url starts", func(t *testing.T) {
		for _, u := range []string{
			"https://auth.example.com",
			"https://auth.example.com/", // trailing slash is normalised away
			"http://localhost:8080",     // local development
		} {
			assert.NoError(t, validateMCPConfig(&config.Config{MCPEnabled: true, AuthorizerURL: u}), u)
		}
	})

	t.Run("MCP enabled without a usable --url refuses to start", func(t *testing.T) {
		for _, u := range []string{
			"",                                 // the shipped default
			"auth.example.com",                 // scheme omitted — the likely mistake
			"ftp://auth.example.com",           // not an http origin
			"https://user:pw@auth.example.com", // userinfo
		} {
			err := validateMCPConfig(&config.Config{MCPEnabled: true, AuthorizerURL: u})
			require.Error(t, err, "--url %q cannot yield a resource identifier, so MCP must not start", u)
			assert.Contains(t, err.Error(), "--url",
				"the message must name the flag an operator has to set")
		}
	})
}
