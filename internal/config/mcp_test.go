package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMCPResource pins the canonical MCP resource identifier.
//
// Three independent things compare against this string — the RFC 9728 metadata
// clients read, the `aud` check in token.ValidateMCPAccessToken, and whatever an
// operator pastes into their client config. A trailing slash or a stray path
// surviving normalization here does not fail loudly; it makes every token's
// audience miss by one character and the surface returns 401 forever.
//
// The empty cases matter just as much: MCPResource returning "" is what makes
// startup refuse --mcp-enabled, so anything that cannot be trusted as a
// canonical origin must return "" rather than a best guess.
func TestMCPResource(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{"plain https origin", "https://auth.example.com", "https://auth.example.com/mcp"},
		{"trailing slash is stripped", "https://auth.example.com/", "https://auth.example.com/mcp"},
		{"path is stripped", "https://auth.example.com/some/base", "https://auth.example.com/mcp"},
		{"query and fragment are stripped", "https://auth.example.com/?a=b#c", "https://auth.example.com/mcp"},
		{"explicit port is part of the origin", "https://auth.example.com:8443", "https://auth.example.com:8443/mcp"},
		{"http is allowed for local development", "http://localhost:8080", "http://localhost:8080/mcp"},
		{"surrounding whitespace is tolerated", "  https://auth.example.com  ", "https://auth.example.com/mcp"},

		{"unset url yields no resource", "", ""},
		{"missing scheme yields no resource", "auth.example.com", ""},
		{"non-http scheme yields no resource", "ftp://auth.example.com", ""},
		{"userinfo yields no resource", "https://user:pass@auth.example.com", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{AuthorizerURL: tc.url}
			assert.Equal(t, tc.want, c.MCPResource())

			// Every self-referential URL the discovery documents publish is
			// built from CanonicalURL, so it must be exactly MCPResource minus
			// the path. Letting them drift is what publishes an authorization
			// server and a jwks_uri that 404 while `resource` looks right.
			if tc.want == "" {
				assert.Equal(t, "", c.CanonicalURL())
			} else {
				assert.Equal(t, tc.want, c.CanonicalURL()+MCPResourcePath)
			}
		})
	}

	t.Run("a nil config yields no resource", func(t *testing.T) {
		var c *Config
		assert.Equal(t, "", c.MCPResource())
		assert.Equal(t, "", c.CanonicalURL())
	})
}
