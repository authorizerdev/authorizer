package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

func setupIntrospectTest(t *testing.T) (*testSetup, string, *token.AuthToken) {
	t.Helper()
	cfg := getTestConfig()
	_, privateKey, publicKey, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	cfg.JWTType = "RS256"
	cfg.JWTPrivateKey = privateKey
	cfg.JWTPublicKey = publicKey
	cfg.JWTSecret = ""
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "introspect_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	authToken, err := ts.TokenProvider.CreateAuthToken(nil, &token.AuthTokenConfig{
		User:        user,
		Roles:       []string{"user"},
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: "basic_auth",
		Nonce:       "nonce-" + uuid.New().String(),
		HostName:    "http://localhost",
	})
	require.NoError(t, err)
	return ts, email, authToken
}

func postIntrospect(t *testing.T, ts *testSetup, form string, basicAuth ...string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.POST("/oauth/introspect", ts.HttpProvider.IntrospectHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/oauth/introspect", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if len(basicAuth) == 2 {
		auth := basicAuth[0] + ":" + basicAuth[1]
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	// Host must match the iss claim baked into tokens at creation time.
	// parsers.GetHost(gc) returns "http://" + req.Host (no X-Forwarded-Proto).
	req.Host = "localhost"
	router.ServeHTTP(w, req)
	return w
}

// formCreds builds a form-encoded body with token + client_id +
// client_secret using the test config defaults.
func formCreds(token, clientID, clientSecret string) string {
	return "token=" + token + "&client_id=" + clientID + "&client_secret=" + clientSecret
}

func TestIntrospectActiveAccessToken(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	form := formCreds(authToken.AccessToken.Token, cfg.ClientID, cfg.ClientSecret)
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["active"], "active access_token must be reported active")
	assert.NotEmpty(t, body["sub"], "active response MUST include sub")
	assert.NotEmpty(t, body["exp"], "active response MUST include exp")
	assert.NotEmpty(t, body["iat"], "active response MUST include iat")
	assert.Equal(t, cfg.ClientID, body["client_id"], "active response MUST include client_id")
	assert.Equal(t, cfg.ClientID, body["aud"], "active response MUST include aud")
}

func TestIntrospectActiveIDToken(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	form := formCreds(authToken.IDToken.Token, cfg.ClientID, cfg.ClientSecret)
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["active"])
}

func TestIntrospectInactiveReturnsOnlyActiveFalse(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := formCreds("this-is-not-a-valid-jwt", cfg.ClientID, cfg.ClientSecret)
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	// RFC 7662 §2.2: inactive response MUST only contain active=false
	assert.Equal(t, false, body["active"])
	assert.Nil(t, body["sub"], "inactive response MUST NOT leak sub")
	assert.Nil(t, body["exp"], "inactive response MUST NOT leak exp")
	assert.Nil(t, body["client_id"], "inactive response MUST NOT leak client_id")
	assert.Nil(t, body["error"], "inactive response MUST NOT contain error")
}

func TestIntrospectMissingTokenReturnsInvalidRequest(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_request", body["error"])
}

func TestIntrospectMissingClientIDReturnsInvalidRequest(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_ = cfg

	form := "token=something"
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntrospectInvalidClientIDReturnsInvalidClient(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	// Wrong client_id, valid secret → must be 401 invalid_client
	// (RFC 6749 §5.2 / RFC 7662 §2.1).
	form := "token=something&client_id=wrong-client-id&client_secret=" + cfg.ClientSecret
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client", body["error"])
}

func TestIntrospectInvalidClientIDViaBasicAuthReturns401(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=something"
	w := postIntrospect(t, ts, form, "wrong-client-id", "wrong-secret")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic", "401 response MUST carry WWW-Authenticate: Basic")
}

func TestIntrospectCacheControlHeaders(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	form := formCreds("anything", cfg.ClientID, cfg.ClientSecret)
	w := postIntrospect(t, ts, form)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
}

// --- M1 / M2 / dead-code regression tests ---

// TestIntrospect_RejectsMissingClientSecret verifies that a form-post
// request providing only client_id (no client_secret) is rejected when
// the server has a client_secret configured. Per RFC 7662 §2.1 client
// authentication is mandatory; allowing client_id alone would defeat
// the purpose of the secret.
func TestIntrospect_RejectsMissingClientSecret(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=something&client_id=" + cfg.ClientID
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusUnauthorized, w.Code, "client_id-only requests must be rejected when a client_secret is configured")
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client", body["error"])
}

// TestIntrospect_RejectsBasicAuthEmptyPassword verifies that an HTTP
// Basic credential with the right client_id but an empty password is
// rejected with 401 invalid_client + a Basic challenge header.
func TestIntrospect_RejectsBasicAuthEmptyPassword(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=something"
	w := postIntrospect(t, ts, form, cfg.ClientID, "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic")
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client", body["error"])
}

// TestIntrospect_RejectsWrongClientSecret verifies that a request with
// the right client_id but a wrong client_secret returns 401
// invalid_client (constant-time comparison).
func TestIntrospect_RejectsWrongClientSecret(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=something&client_id=" + cfg.ClientID + "&client_secret=not-the-right-secret"
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client", body["error"])
}

// TestIntrospect_AcceptsCorrectFormPost verifies the happy path
// (form-post client auth) returns active=true for a valid token.
func TestIntrospect_AcceptsCorrectFormPost(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	form := formCreds(authToken.AccessToken.Token, cfg.ClientID, cfg.ClientSecret)
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["active"])
}

// TestIntrospect_AcceptsCorrectBasicAuth verifies the happy path via
// HTTP Basic client authentication.
func TestIntrospect_AcceptsCorrectBasicAuth(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	form := "token=" + authToken.AccessToken.Token
	w := postIntrospect(t, ts, form, cfg.ClientID, cfg.ClientSecret)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["active"])
}

// TestIntrospect_IgnoresTokenTypeHint verifies that supplying an
// arbitrary token_type_hint produces the same response as omitting it
// (RFC 7662 §2.1 — servers MAY ignore the hint).
func TestIntrospect_IgnoresTokenTypeHint(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	withoutHint := formCreds(authToken.AccessToken.Token, cfg.ClientID, cfg.ClientSecret)
	withHint := withoutHint + "&token_type_hint=refresh_token"

	w1 := postIntrospect(t, ts, withoutHint)
	w2 := postIntrospect(t, ts, withHint)
	require.Equal(t, http.StatusOK, w1.Code)
	require.Equal(t, http.StatusOK, w2.Code)

	var b1, b2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &b1))
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &b2))
	assert.Equal(t, b1["active"], b2["active"])
	assert.Equal(t, b1["sub"], b2["sub"])
	assert.Equal(t, b1["client_id"], b2["client_id"])
}

// TestIntrospect_RateLimited verifies the /oauth/introspect path is
// not exempt from the global rate-limit middleware. The test attaches
// the same RateLimitMiddleware that NewRouter installs, registers the
// introspect handler, then exhausts the burst from a single client IP
// and asserts the next request is throttled with 429. This proves the
// route inherits the same rate-limit treatment as /oauth/token.
func TestIntrospect_RateLimited(t *testing.T) {
	cfg := getTestConfig()
	cfg.RateLimitRPS = 3
	cfg.RateLimitBurst = 3
	ts := initTestSetup(t, cfg)

	w := httptest.NewRecorder()
	_, router := gin.CreateTestContext(w)
	router.Use(ts.HttpProvider.RateLimitMiddleware())
	router.POST("/oauth/introspect", ts.HttpProvider.IntrospectHandler())

	send := func() int {
		req, err := http.NewRequest(http.MethodPost, "/oauth/introspect",
			strings.NewReader("token=x&client_id="+cfg.ClientID+"&client_secret="+cfg.ClientSecret))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "203.0.113.7:1234"
		req.Host = "localhost"
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr.Code
	}

	// Exhaust the burst — these should pass through to the handler
	// (returning 200 active=false because the token is bogus).
	for i := 0; i < 3; i++ {
		code := send()
		assert.NotEqual(t, http.StatusTooManyRequests, code, "burst request %d unexpectedly throttled", i)
	}
	// The next request must be throttled.
	assert.Equal(t, http.StatusTooManyRequests, send(), "request beyond burst MUST be rate-limited")
}

func TestIntrospectDiscoveryAdvertises(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.GET("/.well-known/openid-configuration", ts.HttpProvider.OpenIDConfigurationHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)
	req.Host = "localhost"
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body["introspection_endpoint"], "discovery MUST include introspection_endpoint")
	assert.NotNil(t, body["introspection_endpoint_auth_methods_supported"])
}
