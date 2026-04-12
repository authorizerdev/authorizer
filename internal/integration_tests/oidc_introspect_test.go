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
