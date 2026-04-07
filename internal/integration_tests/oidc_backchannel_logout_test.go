package integration_tests

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/token"
)

// TestBackchannelLogoutSendsLogoutToken verifies that NotifyBackchannelLogout
// POSTs a signed logout_token JWT to the configured URI carrying all the
// claims required by OIDC Back-Channel Logout 1.0 §2.4.
func TestBackchannelLogoutSendsLogoutToken(t *testing.T) {
	cfg := getTestConfig()

	var received atomic.Value // stores the logout_token string
	var contentType atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		contentType.Store(r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		received.Store(form.Get("logout_token"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg.BackchannelLogoutURI = server.URL
	ts := initTestSetup(t, cfg)

	subject := "user-" + uuid.New().String()
	sessionID := "sid-" + uuid.New().String()
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), server.URL, &token.BackchannelLogoutConfig{
		HostName:  "http://localhost",
		Subject:   subject,
		SessionID: sessionID,
	})
	require.NoError(t, err)

	// Poll briefly for the test server to record the token (Notify
	// returns synchronously here, but be defensive).
	deadline := time.Now().Add(2 * time.Second)
	var tokenStr string
	for time.Now().Before(deadline) {
		if v := received.Load(); v != nil {
			tokenStr = v.(string)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NotEmpty(t, tokenStr, "server must have received the logout_token")

	if ct := contentType.Load(); ct != nil {
		assert.Contains(t, ct.(string), "application/x-www-form-urlencoded")
	}

	// Parse the JWT (without signature verification — we trust the signer
	// and only check structural claims here).
	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(tokenStr, claims)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost", claims["iss"])
	assert.Equal(t, cfg.ClientID, claims["aud"])
	assert.Equal(t, subject, claims["sub"])
	assert.Equal(t, sessionID, claims["sid"])
	assert.NotEmpty(t, claims["jti"])
	assert.NotEmpty(t, claims["iat"])

	_, hasNonce := claims["nonce"]
	assert.False(t, hasNonce, "logout_token MUST NOT contain a nonce claim (OIDC BCL 1.0 §2.4)")

	events, ok := claims["events"].(map[string]interface{})
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
