package http_handlers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/token"
)

// recordingTokenProvider is a minimal token.Provider stub used to
// observe arguments passed to NotifyBackchannelLogout. The other
// methods of the interface are intentionally unimplemented — calling
// any of them in a test will panic via the embedded nil interface,
// which is the desired behaviour (it loudly catches accidental usage).
type recordingTokenProvider struct {
	token.Provider // nil embed: only NotifyBackchannelLogout is overridden

	mu        sync.Mutex
	called    bool
	gotConfig *token.BackchannelLogoutConfig
	gotURI    string
	returnErr error
}

func (r *recordingTokenProvider) NotifyBackchannelLogout(_ context.Context, uri string, cfg *token.BackchannelLogoutConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called = true
	r.gotURI = uri
	if cfg != nil {
		c := *cfg
		r.gotConfig = &c
	}
	return r.returnErr
}

// signTestIDToken signs a small id_token-like JWT with HS256 for tests.
func signTestIDToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func newHintTestProvider(_ *testing.T) *httpProvider {
	logger := zerolog.Nop()
	cfg := &config.Config{
		JWTType:   "HS256",
		JWTSecret: "test-secret-do-not-use-in-prod",
	}
	return &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log: &logger,
		},
	}
}

// ---------------------------------------------------------------------------
// id_token_hint binding (H3 + Logic #5)
// ---------------------------------------------------------------------------

func TestLogoutHandler_IDTokenHintAcceptedIfSubMatches(t *testing.T) {
	h := newHintTestProvider(t)
	now := time.Now().Unix()
	hint := signTestIDToken(t, h.Config.JWTSecret, jwt.MapClaims{
		"sub":        "user-123",
		"token_type": "id_token",
		"iat":        now,
		"exp":        now + 60,
	})

	assert.True(t, h.isValidIDTokenHintForSubject(hint, "user-123"),
		"hint with matching sub should be accepted")
}

func TestLogoutHandler_IDTokenHintRejectedIfSubMismatch(t *testing.T) {
	h := newHintTestProvider(t)
	now := time.Now().Unix()
	hint := signTestIDToken(t, h.Config.JWTSecret, jwt.MapClaims{
		"sub":        "attacker-999",
		"token_type": "id_token",
		"iat":        now,
		"exp":        now + 60,
	})

	assert.False(t, h.isValidIDTokenHintForSubject(hint, "victim-123"),
		"hint signed for a different subject must NOT log out the victim (CSRF defence)")
}

func TestLogoutHandler_AcceptsExpiredIDTokenHint(t *testing.T) {
	h := newHintTestProvider(t)
	// Issued and expired well in the past — OIDC Core §3.1.2.1 says
	// expired ID tokens are still valid logout hints provided their
	// signature checks out.
	past := time.Now().Add(-2 * time.Hour).Unix()
	hint := signTestIDToken(t, h.Config.JWTSecret, jwt.MapClaims{
		"sub":        "user-123",
		"token_type": "id_token",
		"iat":        past,
		"exp":        past + 60,
	})

	assert.True(t, h.isValidIDTokenHintForSubject(hint, "user-123"),
		"expired hint with matching sub should still be accepted per OIDC Core §3.1.2.1")
}

func TestLogoutHandler_RejectsBadSignatureHint(t *testing.T) {
	h := newHintTestProvider(t)
	// Sign with a *different* secret.
	now := time.Now().Unix()
	tampered := signTestIDToken(t, "completely-different-secret", jwt.MapClaims{
		"sub":        "user-123",
		"token_type": "id_token",
		"iat":        now,
		"exp":        now + 60,
	})

	assert.False(t, h.isValidIDTokenHintForSubject(tampered, "user-123"),
		"tampered/foreign-signed hint must be rejected")
}

func TestLogoutHandler_RejectsHintWithWrongTokenType(t *testing.T) {
	h := newHintTestProvider(t)
	now := time.Now().Unix()
	hint := signTestIDToken(t, h.Config.JWTSecret, jwt.MapClaims{
		"sub":        "user-123",
		"token_type": "refresh_token",
		"iat":        now,
		"exp":        now + 60,
	})
	assert.False(t, h.isValidIDTokenHintForSubject(hint, "user-123"))
}

func TestLogoutHandler_HintWithoutTokenTypeStillAccepted(t *testing.T) {
	// Some flows do not stamp token_type at all; the helper must
	// still accept those when the sub matches.
	h := newHintTestProvider(t)
	now := time.Now().Unix()
	hint := signTestIDToken(t, h.Config.JWTSecret, jwt.MapClaims{
		"sub": "user-123",
		"iat": now,
		"exp": now + 60,
	})
	assert.True(t, h.isValidIDTokenHintForSubject(hint, "user-123"))
}

func TestLogoutHandler_EmptyExpectedSubjectRejectsHint(t *testing.T) {
	h := newHintTestProvider(t)
	now := time.Now().Unix()
	hint := signTestIDToken(t, h.Config.JWTSecret, jwt.MapClaims{
		"sub":        "user-123",
		"token_type": "id_token",
		"iat":        now,
		"exp":        now + 60,
	})
	// No active session → no expected subject → confirmation must be required.
	assert.False(t, h.isValidIDTokenHintForSubject(hint, ""))
}

func TestLogoutHandler_SecondaryKeyVerifiesHint(t *testing.T) {
	// Manual key rotation: hint was signed with the previous (now
	// secondary) key. Verification should still succeed.
	logger := zerolog.Nop()
	cfg := &config.Config{
		JWTType:            "HS256",
		JWTSecret:          "current-primary-secret",
		JWTSecondaryType:   "HS256",
		JWTSecondarySecret: "previous-secret-still-trusted",
	}
	h := &httpProvider{
		Config:       cfg,
		Dependencies: Dependencies{Log: &logger},
	}

	now := time.Now().Unix()
	hint := signTestIDToken(t, "previous-secret-still-trusted", jwt.MapClaims{
		"sub":        "user-123",
		"token_type": "id_token",
		"iat":        now,
		"exp":        now + 60,
	})
	assert.True(t, h.isValidIDTokenHintForSubject(hint, "user-123"))
}

func TestLogoutHandler_EmptyHintRejected(t *testing.T) {
	h := newHintTestProvider(t)
	assert.False(t, h.isValidIDTokenHintForSubject("", "user-123"))
}

// ---------------------------------------------------------------------------
// Back-channel logout goroutine (H4 + Logic #2 + Logic #11)
// ---------------------------------------------------------------------------

func TestLogoutHandler_BCLGoroutineUsesEmptySessionID(t *testing.T) {
	// Verify the fix for H4: notifyBackchannelLogoutAsync MUST pass an
	// empty SessionID to the token provider. The previous implementation
	// leaked sessionData.Nonce (the in-memory store key) as sid.
	rec := &recordingTokenProvider{}
	logger := zerolog.Nop()
	h := &httpProvider{
		Config: &config.Config{},
		Dependencies: Dependencies{
			Log:           &logger,
			TokenProvider: rec,
		},
	}

	h.notifyBackchannelLogoutAsync(logger, "https://rp.example.com/logout", "https://issuer.example.com", "user-123")

	rec.mu.Lock()
	defer rec.mu.Unlock()
	require.True(t, rec.called, "NotifyBackchannelLogout must be called")
	require.NotNil(t, rec.gotConfig)
	assert.Equal(t, "https://rp.example.com/logout", rec.gotURI)
	assert.Equal(t, "https://issuer.example.com", rec.gotConfig.HostName)
	assert.Equal(t, "user-123", rec.gotConfig.Subject)
	assert.Equal(t, "", rec.gotConfig.SessionID,
		"SessionID MUST be empty — never leak the in-memory nonce as sid")
}

func TestLogoutHandler_BCLGoroutineSwallowsError(t *testing.T) {
	// Smoke test: an error from NotifyBackchannelLogout is logged and
	// swallowed (fire-and-forget). The function must not panic.
	rec := &recordingTokenProvider{returnErr: errors.New("simulated failure")}
	logger := zerolog.Nop()
	h := &httpProvider{
		Config: &config.Config{},
		Dependencies: Dependencies{
			Log:           &logger,
			TokenProvider: rec,
		},
	}
	// Should not panic.
	h.notifyBackchannelLogoutAsync(logger, "https://rp.example.com/logout", "https://issuer.example.com", "user-1")
	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.True(t, rec.called)
}
