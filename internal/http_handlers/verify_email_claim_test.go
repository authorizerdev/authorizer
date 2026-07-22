package http_handlers

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

// VerifyEmailHandler reads the redirect_uri out of a server-signed verification
// token when no redirect_uri query param is supplied. redirect_uri is NOT
// checked by ValidateJWTClaims, so a token minted for a signup/magic-link flow
// that carried no redirect_uri simply has no "redirect_uri" claim. The handler
// previously did claim["redirect_uri"].(string) — a single-return assertion
// that PANICS on an absent (or non-string) claim, crashing the request.
//
// The fix routes the read through the package's claimString helper. These cases
// reproduce the panic-triggering inputs and assert a graceful zero value.
func TestVerifyEmailRedirectClaim_NoPanicOnMissingOrNonString(t *testing.T) {
	// Absent claim (token minted without a redirect_uri).
	require.NotPanics(t, func() {
		require.Equal(t, "", claimString(jwt.MapClaims{}, "redirect_uri"))
	})

	// Present but non-string (e.g. a numeric/bool value in the claim).
	require.NotPanics(t, func() {
		require.Equal(t, "", claimString(jwt.MapClaims{"redirect_uri": 12345}, "redirect_uri"))
	})

	// Present and a string is returned unchanged.
	require.Equal(t, "https://app.example.com/cb",
		claimString(jwt.MapClaims{"redirect_uri": "https://app.example.com/cb"}, "redirect_uri"))
}
