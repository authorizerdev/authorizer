package token

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// signHS256NoKid signs an HMAC token WITHOUT a kid header. Used to
// simulate legacy tokens that were issued before SignJWTToken began
// stamping `kid`, so the verifier exercises the kid-less try-both
// fallback path.
func signHS256NoKid(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// newTestProvider builds a minimal *provider for unit tests using HMAC.
// Caller may override fields on the returned provider's config before
// signing/verifying.
func newTestProvider(t *testing.T) *provider {
	t.Helper()
	logger := zerolog.New(io.Discard)
	return &provider{
		config: &config.Config{
			ClientID:  "test-client",
			JWTType:   "HS256",
			JWTSecret: "primary-secret",
		},
		dependencies: &Dependencies{
			Log: &logger,
		},
	}
}

func TestSignJWTToken_SetsKidHeader(t *testing.T) {
	p := newTestProvider(t)
	tok, err := p.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	require.NoError(t, err)

	parsed, _, err := jwt.NewParser().ParseUnverified(tok, jwt.MapClaims{})
	require.NoError(t, err)
	assert.Equal(t, "test-client", parsed.Header["kid"])
}

func TestParseJWTToken_PrimaryKey(t *testing.T) {
	p := newTestProvider(t)
	tok, err := p.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	require.NoError(t, err)

	claims, err := p.ParseJWTToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
}

func TestParseJWTToken_OptionalIat(t *testing.T) {
	p := newTestProvider(t)
	// Sign without iat — should still parse and validate.
	tok, err := p.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	})
	require.NoError(t, err)

	claims, err := p.ParseJWTToken(tok)
	require.NoError(t, err)
	_, hasIat := claims["iat"]
	assert.False(t, hasIat, "iat must remain absent when not signed in")
}

func TestParseJWTToken_MissingExpRejected(t *testing.T) {
	p := newTestProvider(t)
	tok, err := p.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"iat": time.Now().Unix(),
	})
	require.NoError(t, err)
	// jwt-go validates exp during parse only if present; without exp it
	// passes signature check, then ParseJWTToken's own check fires.
	_, err = p.ParseJWTToken(tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exp")
}

func TestParseJWTToken_LegacyKidlessFallback(t *testing.T) {
	// Legacy token (no kid header) signed under the old secret.
	tok := signHS256NoKid(t, "old-secret", jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})

	verifier := newTestProvider(t)
	verifier.config.JWTSecret = "new-secret"
	verifier.config.JWTSecondaryType = "HS256"
	verifier.config.JWTSecondarySecret = "old-secret"

	claims, err := verifier.ParseJWTToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
}

func TestParseJWTToken_SecondaryKidDirectSelection(t *testing.T) {
	// Manually craft a token with kid = ClientID + "-secondary" to
	// exercise the direct secondary-kid selection path. The token is
	// signed under the secondary secret only — the primary secret
	// would not verify it, so a successful parse proves the kid header
	// drove key selection.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	tok.Header["kid"] = "test-client" + secondaryKidSuffix
	signed, err := tok.SignedString([]byte("old-secret"))
	require.NoError(t, err)

	verifier := newTestProvider(t)
	verifier.config.ClientID = "test-client"
	verifier.config.JWTSecret = "new-secret"
	verifier.config.JWTSecondaryType = "HS256"
	verifier.config.JWTSecondarySecret = "old-secret"

	claims, err := verifier.ParseJWTToken(signed)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
}

func TestParseJWTToken_SecondaryKidWithoutConfig(t *testing.T) {
	// kid claims secondary but verifier has no secondary configured.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	})
	tok.Header["kid"] = "test-client" + secondaryKidSuffix
	signed, err := tok.SignedString([]byte("anything"))
	require.NoError(t, err)

	verifier := newTestProvider(t)
	_, err = verifier.ParseJWTToken(signed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secondary")
}

func TestParseJWTToken_NoFallbackOnMalformed(t *testing.T) {
	verifier := newTestProvider(t)
	verifier.config.JWTSecondaryType = "HS256"
	verifier.config.JWTSecondarySecret = "anything"

	_, err := verifier.ParseJWTToken("not-a-jwt")
	require.Error(t, err)
	// Must NOT report success — fallback only applies to signature errors.
	assert.NotContains(t, err.Error(), "user-1")
}

func TestParseJWTToken_BothKeysFailReturnsWrappedError(t *testing.T) {
	// Sign a token under a third secret that neither primary nor
	// secondary can verify.
	signer := newTestProvider(t)
	signer.config.JWTSecret = "third-secret"
	tok, err := signer.SignJWTToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	require.NoError(t, err)

	verifier := newTestProvider(t)
	verifier.config.JWTSecret = "primary-secret"
	verifier.config.JWTSecondaryType = "HS256"
	verifier.config.JWTSecondarySecret = "secondary-secret"

	_, err = verifier.ParseJWTToken(tok)
	require.Error(t, err)
	assert.True(t, errors.Is(err, jwt.ErrTokenSignatureInvalid), "wrapped error should match jwt.ErrTokenSignatureInvalid via errors.Is")
}

func TestValidateJWTClaims_AudArray(t *testing.T) {
	p := newTestProvider(t)
	claims := jwt.MapClaims{
		"aud":   []interface{}{"other", "test-client"},
		"nonce": "n1",
		"iss":   "https://example.com",
		"sub":   "user-1",
	}
	ok, err := p.ValidateJWTClaims(claims, &AuthTokenConfig{
		Nonce:    "n1",
		HostName: "https://example.com",
		User:     &schemas.User{ID: "user-1"},
	})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestValidateJWTClaims_AudMismatch(t *testing.T) {
	p := newTestProvider(t)
	claims := jwt.MapClaims{
		"aud":   "other-client",
		"nonce": "n1",
		"iss":   "https://example.com",
		"sub":   "user-1",
	}
	_, err := p.ValidateJWTClaims(claims, &AuthTokenConfig{
		Nonce:    "n1",
		HostName: "https://example.com",
		User:     &schemas.User{ID: "user-1"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestValidateJWTClaims_SubMustBeUserIDForOIDCTokens(t *testing.T) {
	p := newTestProvider(t)
	email := "alice@example.com"
	// Access token: email-as-sub must be rejected (OIDC §2 hardening).
	claims := jwt.MapClaims{
		"aud":        "test-client",
		"nonce":      "n1",
		"iss":        "https://example.com",
		"sub":        email,
		"token_type": "access_token",
	}
	_, err := p.ValidateJWTClaims(claims, &AuthTokenConfig{
		Nonce:    "n1",
		HostName: "https://example.com",
		User:     &schemas.User{ID: "user-1", Email: refs.NewStringRef(email)},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject")
}

func TestValidateJWTClaims_VerificationTokenAllowsEmailAsSub(t *testing.T) {
	p := newTestProvider(t)
	email := "alice@example.com"
	// Verification tokens are issued before the user ID exists, so
	// email-as-sub must still be accepted for those token_types.
	claims := jwt.MapClaims{
		"aud":        "test-client",
		"nonce":      "n1",
		"iss":        "https://example.com",
		"sub":        email,
		"token_type": "magic_link_login",
	}
	ok, err := p.ValidateJWTClaims(claims, &AuthTokenConfig{
		Nonce:    "n1",
		HostName: "https://example.com",
		User:     &schemas.User{ID: "", Email: refs.NewStringRef(email)},
	})
	require.NoError(t, err)
	assert.True(t, ok)
}
