package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConsentClientIsLoopbackOnly drives a consent-screen warning both specs ask
// for, so a wrong answer is user-visible: a false negative hides a real risk, a
// false positive cries wolf on a normal web client.
//
// Moved here from internal/clientmetadata when the predicate stopped being a
// property of a metadata document — a client that registered itself via RFC 7591
// needs the identical warning and has no document.
func TestConsentClientIsLoopbackOnly(t *testing.T) {
	cases := []struct {
		name string
		uris []string
		want bool
	}{
		{"all loopback", []string{"http://127.0.0.1/cb", "http://localhost:1/cb"}, true},
		{"ipv6 loopback", []string{"http://[::1]/cb"}, true},
		{"mixed", []string{"http://127.0.0.1/cb", "https://app.example.com/cb"}, false},
		{"public only", []string{"https://app.example.com/cb"}, false},
		{"empty", nil, false},
		{"lookalike host is not loopback", []string{"https://127.0.0.1.evil.com/cb"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, consentClient{RedirectURIs: tc.uris}.isLoopbackOnly())
		})
	}
}

// TestValidateRegistrationRedirectURI pins what an anonymous caller may store as
// a redirect target. Every rejection here is a place a code could otherwise be
// delivered somewhere the user did not intend.
func TestValidateRegistrationRedirectURI(t *testing.T) {
	cases := []struct {
		name string
		uri  string
		ok   bool
		why  string
	}{
		{"https", "https://app.example.com/cb", true, "the normal case"},
		{"loopback http", "http://127.0.0.1:5599/cb", true, "RFC 8252 §7.3 native clients"},
		{"localhost http", "http://localhost:1234/cb", true, "same, by name"},
		{"ipv6 loopback http", "http://[::1]:1234/cb", true, "same, over IPv6"},
		{"public http", "http://app.example.com/cb", false, "codes would cross the network in the clear"},
		{"custom scheme", "myapp://cb", false, "the server cannot tell which local app receives the code"},
		{"fragment", "https://app.example.com/cb#f", false, "RFC 6749 §3.1.2 forbids a fragment"},
		{"relative", "/cb", false, "a redirect_uri must be absolute"},
		{"empty", "", false, "nothing to redirect to"},
		{"no host", "https:///cb", false, "an https URI with no host is not a destination"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRegistrationRedirectURI(tc.uri)
			if tc.ok {
				assert.NoError(t, err, tc.why)
				return
			}
			assert.Error(t, err, tc.why)
		})
	}
}
