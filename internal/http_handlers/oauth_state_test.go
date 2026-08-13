package http_handlers

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAuthStateHandleCarriesNothingFromTheCaller is the regression guard for a
// field-shifting bug in the old "___"-joined state.
//
// The caller supplied the FIRST field, so any delimiter it contained moved every
// later field left — handing the callback its redirect URI and its ROLES from
// caller input. The callback validates roles only against ProtectedRoles, so an
// injected role that is merely non-protected was accepted.
//
// The property now is stronger than "the delimiter is escaped": the caller's
// value never travels to the provider at all. The handle is opaque and the
// parameters are read from the store, so each state below — every one of which
// broke the old format — cannot influence anything.
func TestOAuthStateHandleCarriesNothingFromTheCaller(t *testing.T) {
	const (
		provider    = "google"
		redirectURI = "http://authorizer:8080/app"
		roles       = "user"
		scope       = "openid profile email"
	)

	for _, callerState := range []string{
		"UlFosaMoufBG7UKydMfQJDscQHkYwZA71kGGQgFmN",
		// ~1 in 64 base64url tokens ends in "_". Merged with the old "___" it
		// produced "____", split a character early, and left the redirect URI
		// as "_http://..." — a hard failure on every provider.
		"UlFosaMoufBG7UKydMfQJDscQHkYwZA71kGGQgFmN_",
		"abc___def",
		// The malicious shape: forging every field that follows.
		"A___https://evil.example/steal___admin___openid",
		"______",
		"",
		"état.测试.🔐",
		"a.b.c.d.e",
	} {
		t.Run(callerState, func(t *testing.T) {
			handle, err := newOAuthStateHandle()
			require.NoError(t, err)

			// The handle is what the provider sees. It must contain nothing of
			// the caller's, whatever they sent. (An empty state has no prefix to
			// look for — there is nothing it could have leaked.)
			if len(callerState) >= 8 {
				assert.NotContains(t, handle, callerState[:8],
					"the caller's value must not appear in the value sent to the provider")
			}

			raw, err := marshalOAuthState(oauthStatePayload{
				Provider: provider, State: callerState,
				RedirectURI: redirectURI, Roles: roles, Scope: scope,
			})
			require.NoError(t, err)

			got, err := unmarshalOAuthState(raw)
			require.NoError(t, err)

			assert.Equal(t, callerState, got.State, "the caller's state must round-trip unchanged")
			assert.Equal(t, redirectURI, got.RedirectURI,
				"the redirect URI must come from the server, never from the caller")
			assert.Equal(t, roles, got.Roles,
				"roles must come from the server, never from the caller")
			assert.Equal(t, scope, got.Scope)
			assert.Equal(t, provider, got.Provider)
		})
	}
}

// TestOAuthStateHandleIsOpaqueAndShort pins the two properties that made a
// handle preferable to encoding the fields inline.
//
// X/Twitter documents a 100-character limit on `state`, and a realistic
// redirect URI already pushed the old concatenated form past it. Encoding each
// field instead would have closed the injection but inflated the value by ~28%,
// making that worse. A fixed-size handle is shorter than what this server sent
// before, whatever the caller's state and redirect URI happen to be.
func TestOAuthStateHandleIsOpaqueAndShort(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		h, err := newOAuthStateHandle()
		require.NoError(t, err)

		assert.LessOrEqual(t, len(h), 64, "must stay well inside the tightest provider limit (X/Twitter: 100)")
		assert.False(t, seen[h], "handles must not repeat")
		seen[h] = true

		// URL-safe and unreserved (RFC 3986 §2.3), so no provider has to
		// percent-encode it and hand back something subtly different.
		for _, r := range h {
			assert.True(t,
				(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
					(r >= '0' && r <= '9') || r == '-' || r == '_',
				"unexpected character %q in handle", r)
		}
		_, err = base64.RawURLEncoding.DecodeString(h)
		assert.NoError(t, err, "handle must be raw base64url")
	}
}

// TestOAuthStateRejectsForeignEntries pins that the callback fails closed on a
// store entry it did not write — including one left by the previous release,
// whose value was the bare provider name rather than JSON. Guessing at such an
// entry is precisely how caller input became privileges.
func TestOAuthStateRejectsForeignEntries(t *testing.T) {
	for _, tc := range []struct{ name, raw string }{
		{"empty", ""},
		{"previous release's bare provider name", "google"},
		{"the old ___ format", "tok___http://x/app___user___openid"},
		{"json without a provider", `{"state":"x","redirect_uri":"http://x/app"}`},
		{"not json", "!!!not-json!!!"},
		{"json array", `["google","x"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := unmarshalOAuthState(tc.raw)
			assert.ErrorIs(t, err, errMalformedOAuthState)
		})
	}
}
