package http_handlers

import (
	"github.com/authorizerdev/authorizer/internal/crypto"
)

// The memory-store session entry IS the revocation record in this codebase.
// Logout, password reset, admin session-wipe and /oauth/revoke all revoke by
// deleting it, and nothing else records revocation anywhere. Every surface that
// needs to answer "is this token still live?" therefore has to read it:
// token.validateStatefulAccessToken does so on every authenticated request, and
// the two token-facing endpoints in this package do so through the helpers here.
//
// They exist as one definition rather than two inline copies because
// /oauth/introspect and /oauth/revoke now ask exactly the same question, and a
// second copy is a second place for the answer to drift.

// sessionEntryMatches reports whether the memory store still holds this exact
// token under the given entry key.
//
// The VALUE is compared, not merely the entry's existence, to stay identical to
// token.validateStatefulAccessToken — one definition of "this token is the live
// one", not two.
//
// Note it does NOT demote a merely superseded token: the refresh grant mints a
// fresh nonce and leaves the previous nonce's entries alone, so a still-unexpired
// predecessor legitimately keeps its own live entry until it expires.
func (h *httpProvider) sessionEntryMatches(sessionKey, entryKey, presented string) bool {
	stored, err := h.MemoryStoreProvider.GetUserSession(sessionKey, entryKey)
	if err != nil {
		return false
	}
	// Dual-read against the stored digest — see crypto.VerifySessionValue. The
	// client_credentials path writes the raw token rather than a digest, which
	// that function's legacy branch handles.
	return crypto.VerifySessionValue(presented, stored)
}

// claimsSessionKey derives the memory-store session key a token addresses.
// Mirrors token.validateStatefulAccessToken and RevokeRefreshTokenHandler: the
// shape is "<login_method>:<sub>", or a bare "<sub>" when the token carries no
// login_method.
func claimsSessionKey(claims map[string]interface{}) string {
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return ""
	}
	if lm, _ := claims["login_method"].(string); lm != "" {
		return lm + ":" + sub
	}
	return sub
}
