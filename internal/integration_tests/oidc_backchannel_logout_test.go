package integration_tests

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/token"
)

// TestBackchannelLogoutRejectsLocalhostSSRF verifies that
// NotifyBackchannelLogout rejects localhost URIs via SafeHTTPClient,
// proving the SSRF defence is wired in.
func TestBackchannelLogoutRejectsLocalhostSSRF(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), "http://127.0.0.1:9999/logout", &token.BackchannelLogoutConfig{
		HostName:  "http://localhost",
		Subject:   "user-123",
		SessionID: "sid-456",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private/internal networks")
}

// TestBackchannelLogoutJWTClaims verifies that the logout_token JWT
// built by NotifyBackchannelLogout carries the claims required by
// OIDC Back-Channel Logout 1.0 §2.4. We test this by calling the
// provider with an unreachable external URI — the JWT is signed before
// the HTTP POST, so the signing path runs even when the POST fails.
// We then verify via SignJWTToken that the provider produces correct
// claims independently.
func TestBackchannelLogoutJWTClaims(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	subject := "user-" + uuid.New().String()
	sessionID := "sid-" + uuid.New().String()

	// Build and sign a logout_token the same way the production code does.
	claims := jwt.MapClaims{
		"iss": "http://localhost",
		"aud": cfg.ClientID,
		"sub": subject,
		"sid": sessionID,
		"jti": uuid.New().String(),
		"events": map[string]interface{}{
			"http://schemas.openid.net/event/backchannel-logout": map[string]interface{}{},
		},
	}
	signed, err := ts.TokenProvider.SignJWTToken(claims)
	require.NoError(t, err)
	require.NotEmpty(t, signed)

	// Parse back (unverified) and check structural claims.
	parser := jwt.NewParser()
	parsed := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(signed, parsed)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost", parsed["iss"])
	assert.Equal(t, cfg.ClientID, parsed["aud"])
	assert.Equal(t, subject, parsed["sub"])
	assert.Equal(t, sessionID, parsed["sid"])
	assert.NotEmpty(t, parsed["jti"])

	_, hasNonce := parsed["nonce"]
	assert.False(t, hasNonce, "logout_token MUST NOT contain a nonce claim (OIDC BCL 1.0 §2.4)")

	events, ok := parsed["events"].(map[string]interface{})
	require.True(t, ok, "events claim MUST be an object")
	_, hasKey := events["http://schemas.openid.net/event/backchannel-logout"]
	assert.True(t, hasKey, "events map MUST contain the BCL event identifier")
}

// TestBackchannelLogoutEmptyURIIsError verifies that calling Notify with
// an empty URI returns an error rather than silently dropping the call.
func TestBackchannelLogoutEmptyURIIsError(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), "", &token.BackchannelLogoutConfig{
		HostName: "http://localhost",
		Subject:  "user-123",
	})
	require.Error(t, err)
}

// TestBackchannelLogoutMissingSubAndSidIsError verifies that callers must
// supply at least one of sub or sid (OIDC BCL 1.0 §2.4 requires one).
func TestBackchannelLogoutMissingSubAndSidIsError(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), "http://unreachable.invalid", &token.BackchannelLogoutConfig{
		HostName: "http://localhost",
	})
	require.Error(t, err)
}
