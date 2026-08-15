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

	"github.com/authorizerdev/authorizer/internal/constants"
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
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		Nonce:       "nonce-" + uuid.New().String(),
		HostName:    "http://localhost",
	})
	require.NoError(t, err)

	// Mirror the real login flow: persist the session + access token in the
	// memory store. Introspection reports a token whose session entry is gone as
	// inactive (RFC 7662 §2.2), so a token minted without one is a REVOKED token,
	// not a live one — this helper used to hand every test that shape and then
	// assert it was active.
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		introspectSessionKey(user.ID),
		constants.TokenTypeSessionToken+"_"+authToken.FingerPrint,
		authToken.FingerPrintHash,
		authToken.SessionTokenExpiresAt,
	))
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		introspectSessionKey(user.ID),
		constants.TokenTypeAccessToken+"_"+authToken.FingerPrint,
		authToken.AccessToken.Token,
		authToken.AccessToken.ExpiresAt,
	))

	return ts, email, authToken
}

// introspectSessionKey is the memory-store session key for a basic_auth login,
// matching what token.validateStatefulAccessToken derives from the claims.
func introspectSessionKey(userID string) string {
	return constants.AuthRecipeMethodBasicAuth + ":" + userID
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

func TestIntrospectActiveAccessToken(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	form := "token=" + authToken.AccessToken.Token + "&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret
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

// TestIntrospectActiveIDToken is the guard on introspection's id_token carve-out,
// not merely a happy-path case. An id_token is never registered in the memory
// store — it is an assertion, not a credential, and nothing revokes one — so the
// session-liveness check in tokenSessionIsLive skips it by token type. Widen that
// check to every token type and this test is what fails; do not delete it as a
// duplicate of TestIntrospectActiveAccessToken.
func TestIntrospectActiveIDToken(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	form := "token=" + authToken.IDToken.Token + "&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["active"])
}

// TestIntrospectRevokedSessionIsInactive is the regression test for the RFC 7662
// §2.2 gap: introspection validated signature, exp, iss, aud and the user's
// RevokedTimestamp, but never the memory-store session entry — which is the only
// place revocation is recorded. Logout, password reset, admin session-wipe and
// /oauth/revoke all revoke by deleting it, so a token Authorizer's own API
// rejected still introspected as active, disclosing sub/scope/aud to any resource
// server that trusted the answer.
func TestIntrospectRevokedSessionIsInactive(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config

	claims, err := ts.TokenProvider.ParseJWTToken(authToken.AccessToken.Token)
	require.NoError(t, err)
	userID, _ := claims["sub"].(string)
	require.NotEmpty(t, userID)

	form := "token=" + authToken.AccessToken.Token + "&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret

	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	var before map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &before))
	require.Equal(t, true, before["active"], "baseline: a live token must introspect as active")

	// Revoke exactly the way logout and RevokeRefreshTokenHandler do.
	require.NoError(t, ts.MemoryStoreProvider.DeleteUserSession(introspectSessionKey(userID), authToken.FingerPrint))

	// The token is now dead at Authorizer's own API surface...
	vReq, _ := http.NewRequest(http.MethodGet, "/userinfo", nil)
	vReq.Host = "localhost"
	_, vErr := ts.TokenProvider.ValidateAccessToken(&gin.Context{Request: vReq}, authToken.AccessToken.Token)
	require.Error(t, vErr, "baseline: the revoked token must no longer validate")

	// ...so introspection must not report it live either.
	w = postIntrospect(t, ts, form)
	require.Equal(t, http.StatusOK, w.Code)
	var after map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &after))
	assert.Equal(t, false, after["active"], "a revoked token MUST introspect as inactive (RFC 7662 §2.2)")
	assert.Nil(t, after["sub"], "inactive response MUST NOT leak sub")
	assert.Nil(t, after["scope"], "inactive response MUST NOT leak scope")
}

func TestIntrospectInactiveReturnsOnlyActiveFalse(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=this-is-not-a-valid-jwt&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret
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

	form := "token=something"
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntrospectInvalidClientIDReturnsInvalidClient(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=something&client_id=wrong-client-id"
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusBadRequest, w.Code)
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
	form := "token=anything&client_id=" + cfg.ClientID + "&client_secret=" + cfg.ClientSecret
	w := postIntrospect(t, ts, form)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
}

func TestIntrospectMissingClientSecretRejectsWhenConfigured(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	// Server has ClientSecret configured; omitting it must be rejected.
	form := "token=anything&client_id=" + cfg.ClientID
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client", body["error"])
}

func TestIntrospectWrongClientSecretRejects(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	form := "token=anything&client_id=" + cfg.ClientID + "&client_secret=wrong-secret"
	w := postIntrospect(t, ts, form)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_client", body["error"])
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
