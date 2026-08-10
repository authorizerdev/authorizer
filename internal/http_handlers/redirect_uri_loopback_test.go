package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRedirectURIMatches pins both halves of RFC 8252 §7.3: loopback redirects
// must tolerate any port, and nothing else may.
//
// The negative cases carry the weight. This function widens redirect_uri
// validation, which is the check standing between a registered client and an
// authorization code being delivered to an attacker's endpoint, so every way the
// carve-out could leak past loopback is asserted explicitly.
func TestRedirectURIMatches(t *testing.T) {
	cases := []struct {
		name       string
		registered string
		presented  string
		want       bool
	}{
		{"identical", "https://app.example.com/cb", "https://app.example.com/cb", true},

		// RFC 8252 §7.3: a native app binds an ephemeral port at run time.
		{"loopback ignores an added port", "http://127.0.0.1/callback", "http://127.0.0.1:53119/callback", true},
		{"loopback ignores a changed port", "http://127.0.0.1:1/callback", "http://127.0.0.1:53119/callback", true},
		{"localhost ignores a port", "http://localhost/callback", "http://localhost:3118/callback", true},
		{"ipv6 loopback ignores a port", "http://[::1]/callback", "http://[::1]:8080/callback", true},

		// The carve-out must not reach anything that is not loopback.
		{"public host keeps exact port matching", "https://app.example.com/cb", "https://app.example.com:8443/cb", false},
		{"public host does not match loopback", "https://app.example.com/cb", "http://127.0.0.1:9/cb", false},
		{"loopback does not match a public host", "http://127.0.0.1/cb", "https://attacker.example.com/cb", false},

		// Only the port is ignored — never scheme, host, path or query.
		{"path must still match", "http://127.0.0.1/callback", "http://127.0.0.1:9/evil", false},
		{"scheme must still match", "http://127.0.0.1/callback", "https://127.0.0.1:9/callback", false},
		{"query must still match", "http://127.0.0.1/cb?a=1", "http://127.0.0.1:9/cb?a=2", false},
		{"localhost and 127.0.0.1 are different names", "http://localhost/cb", "http://127.0.0.1:9/cb", false},

		// A hostname that merely contains a loopback name is not loopback.
		{"lookalike host is not loopback", "http://127.0.0.1/cb", "http://127.0.0.1.evil.com:9/cb", false},
		{"localhost subdomain is not loopback", "http://localhost/cb", "http://localhost.evil.com:9/cb", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, redirectURIMatches(tc.registered, tc.presented))
		})
	}
}
