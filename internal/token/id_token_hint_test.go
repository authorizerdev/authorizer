package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIDTokenHint_AcceptsExpired(t *testing.T) {
	p := newTestProvider(t)
	tok, err := p.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // expired
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})
	require.NoError(t, err)

	// Sanity: ParseJWTToken would reject as expired (jwt-go enforces exp).
	_, parseErr := p.ParseJWTToken(tok)
	require.Error(t, parseErr)

	// Hint accepts the expired token.
	claims, err := p.ParseIDTokenHint(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
}

func TestParseIDTokenHint_RejectsBadSignature(t *testing.T) {
	signer := newTestProvider(t)
	signer.config.JWTSecret = "different-secret"
	tok, err := signer.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)

	verifier := newTestProvider(t)
	// verifier uses "primary-secret" by default, so signature won't match.
	_, err = verifier.ParseIDTokenHint(tok)
	require.Error(t, err)
}

func TestParseIDTokenHint_RejectsMalformed(t *testing.T) {
	p := newTestProvider(t)
	_, err := p.ParseIDTokenHint("not.a.jwt")
	require.Error(t, err)
}

func TestParseIDTokenHint_EmptyToken(t *testing.T) {
	p := newTestProvider(t)
	_, err := p.ParseIDTokenHint("")
	require.Error(t, err)
}

func TestParseIDTokenHint_SecondaryKidSelection(t *testing.T) {
	// Expired token signed under secondary secret with kid pointing
	// at the secondary key. ParseIDTokenHint should accept it.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	tok.Header["kid"] = "test-client" + secondaryKidSuffix
	signed, err := tok.SignedString([]byte("old-secret"))
	require.NoError(t, err)

	verifier := newTestProvider(t)
	verifier.config.JWTSecret = "new-secret"
	verifier.config.JWTSecondaryType = "HS256"
	verifier.config.JWTSecondarySecret = "old-secret"

	claims, err := verifier.ParseIDTokenHint(signed)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
}

func TestParseIDTokenHint_LegacyKidlessFallback(t *testing.T) {
	// Legacy expired token (no kid) signed under old secret. The
	// kid-less try-both fallback inside ParseIDTokenHint should
	// route to the secondary key.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte("old-secret"))
	require.NoError(t, err)

	verifier := newTestProvider(t)
	verifier.config.JWTSecret = "new-secret"
	verifier.config.JWTSecondaryType = "HS256"
	verifier.config.JWTSecondarySecret = "old-secret"

	claims, err := verifier.ParseIDTokenHint(signed)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
}
