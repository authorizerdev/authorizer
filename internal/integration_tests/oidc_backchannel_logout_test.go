package integration_tests

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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

	// NotifyBackchannelLogout is invoked synchronously below, so we
	// can capture the request inline without polling.
	var receivedToken string
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		receivedToken = form.Get("logout_token")
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

	require.NotEmpty(t, receivedToken, "server must have received the logout_token")
	assert.Contains(t, receivedContentType, "application/x-www-form-urlencoded")

	// Parse the JWT (without signature verification — we trust the signer
	// and only check structural claims here).
	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(receivedToken, claims)
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
// an empty URI returns ErrBackchannelURIEmpty rather than silently
// dropping the call.
func TestBackchannelLogoutEmptyURIIsError(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), "", &token.BackchannelLogoutConfig{
		HostName: "http://localhost",
		Subject:  "user-123",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, token.ErrBackchannelURIEmpty), "expected ErrBackchannelURIEmpty, got %v", err)
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
	assert.True(t, errors.Is(err, token.ErrBackchannelMissingSubAndSid))
}

// TestBackchannelLogoutMissingHostNameIsError verifies that callers must
// supply a HostName for the iss claim.
func TestBackchannelLogoutMissingHostNameIsError(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), "http://example.test", &token.BackchannelLogoutConfig{
		Subject: "user-123",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, token.ErrBackchannelMissingHostName))
}

// TestBackchannelLogoutStatusCheck verifies that a non-2xx response from
// the receiver is surfaced as an error per OIDC BCL 1.0 §2.8.
func TestBackchannelLogoutStatusCheck(t *testing.T) {
	cfg := getTestConfig()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), server.URL, &token.BackchannelLogoutConfig{
		HostName:  "http://localhost",
		Subject:   "user-1",
		SessionID: "sid-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	// Error must not contain the full URL — only the host.
	assert.NotContains(t, err.Error(), server.URL)
}

// TestBackchannelLogoutSuccess verifies that a 2xx response (here 204
// No Content) is treated as success and the body is drained without
// error.
func TestBackchannelLogoutSuccess(t *testing.T) {
	cfg := getTestConfig()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), server.URL, &token.BackchannelLogoutConfig{
		HostName:  "http://localhost",
		Subject:   "user-1",
		SessionID: "sid-1",
	})
	require.NoError(t, err)
}

// TestBackchannelLogoutOmitsSidWhenEmpty verifies that the sid claim
// is omitted entirely when SessionID is empty (OIDC BCL 1.0 §2.4 makes
// sid OPTIONAL — Branch 5 of the logout flow relies on this contract).
func TestBackchannelLogoutOmitsSidWhenEmpty(t *testing.T) {
	cfg := getTestConfig()
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		receivedToken = form.Get("logout_token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), server.URL, &token.BackchannelLogoutConfig{
		HostName: "http://localhost",
		Subject:  "user-no-sid",
		// SessionID intentionally empty.
	})
	require.NoError(t, err)
	require.NotEmpty(t, receivedToken)

	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(receivedToken, claims)
	require.NoError(t, err)
	_, hasSid := claims["sid"]
	assert.False(t, hasSid, "sid claim must be omitted when SessionID is empty")
	assert.Equal(t, "user-no-sid", claims["sub"])
}

// TestBackchannelLogoutIncludesSidWhenSet verifies that the sid claim
// is present when SessionID is non-empty.
func TestBackchannelLogoutIncludesSidWhenSet(t *testing.T) {
	cfg := getTestConfig()
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		receivedToken = form.Get("logout_token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), server.URL, &token.BackchannelLogoutConfig{
		HostName:  "http://localhost",
		Subject:   "user-with-sid",
		SessionID: "session-xyz",
	})
	require.NoError(t, err)
	require.NotEmpty(t, receivedToken)

	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(receivedToken, claims)
	require.NoError(t, err)
	assert.Equal(t, "session-xyz", claims["sid"])
}

// TestBackchannelLogoutRejectsLoopback verifies that the SSRF filter is
// engaged when the test bypass is disabled. We force the bypass off for
// this test only and point at a loopback URL — SafeHTTPClient must
// reject it before any HTTP I/O happens.
func TestBackchannelLogoutRejectsLoopback(t *testing.T) {
	cfg := getTestConfig()
	// Disable the test SSRF bypass for this test only so SafeHTTPClient
	// runs against an httptest loopback URL and rejects it.
	cfg.SkipTestEndpointSSRFValidation = false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server must NOT be reached when SSRF filter is on")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts := initTestSetup(t, cfg)
	err := ts.TokenProvider.NotifyBackchannelLogout(context.Background(), server.URL, &token.BackchannelLogoutConfig{
		HostName:  "http://localhost",
		Subject:   "user-1",
		SessionID: "sid-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF filter")
}
