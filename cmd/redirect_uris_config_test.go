package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

// TestValidateRedirectURIsRejectsUnmatchableValues pins that a --redirect-uris
// entry which could never match is refused at boot rather than at every login.
//
// The comparison this list feeds is exact, so an unusable entry does not fail
// loudly on its own: the flag looks configured, the callback is refused with
// "invalid redirect_uri", and nothing names the entry at fault.
func TestValidateRedirectURIsRejectsUnmatchableValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		uris []string
		ok   bool
	}{
		{"unset", nil, true},
		{"blank entries are skipped", []string{"", "   "}, true},
		{"absolute https", []string{"https://app.example.com/callback"}, true},
		{"loopback http", []string{"http://127.0.0.1:8080/callback"}, true},
		{"several", []string{"https://a.example.com/cb", "https://b.example.com/cb"}, true},

		{"no scheme", []string{"app.example.com/callback"}, false},
		{"non-http scheme", []string{"myapp://callback"}, false},
		{"no host", []string{"https:///callback"}, false},
		// Rejected for the same reasons redirectURIMatches rejects them: a
		// fragment swallows the authorization response, user info reads as one
		// host while resolving to another.
		{"fragment", []string{"https://app.example.com/cb#x"}, false},
		{"user info", []string{"https://evil.example.com@app.example.com/cb"}, false},
		{"one bad entry fails the whole list", []string{"https://ok.example.com/cb", "nope"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedirectURIs(&config.Config{RedirectURIs: tc.uris})
			if tc.ok {
				require.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}
