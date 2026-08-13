package cmd

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// TestValidateAuthorizerURLRequiresAUsableValue pins that the shipped binary
// cannot start in the header-derived-host configuration.
//
// Without --url the password-reset link, the email-verification link, the magic
// link and the JWT `iss` claim all come from request headers, so an
// unauthenticated attacker can have a victim emailed a genuine reset link
// pointing at a domain the attacker controls, then redeem the harvested token by
// replaying the same spoofed Host (CWE-640).
func TestValidateAuthorizerURLRequiresAUsableValue(t *testing.T) {
	t.Run("empty is refused", func(t *testing.T) {
		err := validateAuthorizerURL(&config.Config{AuthorizerURL: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--url is required")
	})

	t.Run("whitespace-only is refused", func(t *testing.T) {
		require.Error(t, validateAuthorizerURL(&config.Config{AuthorizerURL: "   "}))
	})

	// The subtle half. SetTrustedURL treats anything it cannot normalize as
	// UNSET and silently resumes header derivation, so a value that merely looks
	// configured would start the server in the vulnerable state. Rejecting only
	// the empty string would leave that door open.
	t.Run("unusable values are refused, not silently downgraded", func(t *testing.T) {
		for _, bad := range []string{
			"auth.example.com",          // no scheme
			"//auth.example.com",        // scheme-relative
			"ftp://auth.example.com",    // wrong scheme
			"javascript:alert(1)",       // not a location at all
			"https://",                  // no host
			"https://user:pw@auth.test", // user info
			"not a url",                 //
		} {
			t.Run(bad, func(t *testing.T) {
				require.Equal(t, "", parsers.SanitizeAuthorizerURL(bad),
					"precondition: this value must be one SetTrustedURL would discard")
				err := validateAuthorizerURL(&config.Config{AuthorizerURL: bad})
				require.Error(t, err, "an unusable --url must not be accepted")
				assert.Contains(t, err.Error(), "not a usable canonical URL")
			})
		}
	})

	t.Run("usable values are accepted", func(t *testing.T) {
		for _, ok := range []string{
			"https://auth.example.com",
			"http://localhost:8080",
			"https://auth.example.com:8443",
			"https://auth.example.com/", // trailing slash normalises away
			"  https://auth.example.com  ",
		} {
			t.Run(ok, func(t *testing.T) {
				assert.NoError(t, validateAuthorizerURL(&config.Config{AuthorizerURL: ok}))
			})
		}
	})
}

// TestTrustedURLBeatsEveryHeader is the property the requirement buys: once a
// usable --url is set, no request header can influence the host this server
// considers its own. Without this, requiring --url would be bookkeeping.
func TestTrustedURLBeatsEveryHeader(t *testing.T) {
	parsers.SetTrustedURL("https://auth.example.com")
	t.Cleanup(func() { parsers.SetTrustedURL("") })

	req := httptest.NewRequest("GET", "http://ignored.test/forgot_password", nil)
	req.Host = "evil.example"
	req.Header.Set("X-Forwarded-Host", "evil.example")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Authorizer-URL", "https://evil.example")

	assert.Equal(t, "https://auth.example.com", parsers.GetHostFromRequest(req),
		"a configured canonical URL must win over X-Authorizer-URL, X-Forwarded-Host and Host")
}

// TestHeaderDerivationStillWorksWhenUnset guards the fallback that library
// embedders and the test suite still rely on. The shipped binary can no longer
// reach it, but it must not rot: silently returning "" here would break callers
// that construct a Config directly instead of going through cobra.
func TestHeaderDerivationStillWorksWhenUnset(t *testing.T) {
	parsers.SetTrustedURL("")

	req := httptest.NewRequest("GET", "http://ignored.test/", nil)
	req.Host = "embedded.test"
	assert.Equal(t, "http://embedded.test", parsers.GetHostFromRequest(req))
}
