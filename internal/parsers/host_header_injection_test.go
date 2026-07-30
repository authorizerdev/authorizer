package parsers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Verification of GHSA-m82j-rq33-qjx2 (Password Reset Poisoning via Host Header
// Injection, CWE-640).
//
// GetHostFromRequest is the single point the advisory's whole data flow hangs
// off: internal/service/forgot_password.go takes `meta.HostURL` from it, uses it
// to build the emailed reset link and the JWT `iss`, and
// internal/service/reset_password.go re-validates that `iss` against the SAME
// function's output on the redemption request. So whether the attack works is
// decided entirely here.
//
// These tests pin BOTH halves of the current state: the mitigation works when
// --url is set, and the vulnerable fallback is still live when it is not.
func TestHostHeaderInjection_GHSA_m82j_rq33_qjx2(t *testing.T) {
	// Package-level trustedURL is set once at startup; restore it so these
	// subtests cannot leak into each other or into other tests in this package.
	original := trustedURL
	t.Cleanup(func() { trustedURL = original })

	poisoned := func() *http.Request {
		r, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/graphql", nil)
		r.Host = "attacker.example"
		r.Header.Set("X-Forwarded-Host", "attacker.example")
		r.Header.Set("X-Forwarded-Proto", "https")
		return r
	}

	t.Run("MITIGATED: --url set makes every attacker header irrelevant", func(t *testing.T) {
		SetTrustedURL("https://auth.example.com")

		got := GetHostFromRequest(poisoned())
		assert.Equal(t, "https://auth.example.com", got,
			"with --url configured the trusted value must win over Host/X-Forwarded-Host")

		// X-Authorizer-URL is the third header the fallback path honours; it must
		// be ignored too, not just the forwarding headers.
		r := poisoned()
		r.Header.Set("X-Authorizer-URL", "https://attacker.example")
		assert.Equal(t, "https://auth.example.com", GetHostFromRequest(r),
			"X-Authorizer-URL must not override the configured --url either")
	})

	t.Run("VULNERABLE: --url unset lets the attacker choose the reset-link host", func(t *testing.T) {
		SetTrustedURL("") // the shipped default (cmd/root.go: --url defaults to "")

		got := GetHostFromRequest(poisoned())

		// This is the advisory's core finding, asserted as current behaviour
		// rather than as the desired one. forgot_password builds
		//   redirectURI = hostname + "/app/reset-password"
		// from exactly this value, so the emailed link points at the attacker.
		assert.Equal(t, "https://attacker.example", got,
			"documents GHSA-m82j-rq33-qjx2: with --url unset the host is attacker-controlled")

		// sanitizeHost only strips structural characters — it applies no
		// allowlist, so an arbitrary registrable domain passes untouched.
		r := poisoned()
		r.Header.Set("X-Forwarded-Host", "evil.co.uk")
		assert.Equal(t, "https://evil.co.uk", GetHostFromRequest(r),
			"sanitizeHost performs no allowlist check, only structural filtering")
	})

	t.Run("sanitizer still blocks structural injection regardless of --url", func(t *testing.T) {
		SetTrustedURL("")
		for _, bad := range []string{
			"attacker.example/path",
			"attacker.example?q=1",
			"attacker.example#frag",
			"user@attacker.example",
			"attacker.example\r\nX-Injected: 1",
		} {
			r, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/graphql", nil)
			r.Header.Set("X-Forwarded-Host", bad)
			r.Host = "real.example"
			got := GetHostFromRequest(r)
			assert.Equal(t, "http://real.example", got,
				"a structurally malformed X-Forwarded-Host (%q) must be rejected and fall through", bad)
		}
	})
}
