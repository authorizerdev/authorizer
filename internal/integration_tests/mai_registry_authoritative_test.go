package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// addServiceAccountClient inserts an active service_account client into the
// registry with a bcrypt-hashed secret and returns its client_id + plaintext
// secret. Mirrors the reserved-client seed hashing (cost 12).
func addServiceAccountClient(t *testing.T, ts *testSetup, secret string) string {
	t.Helper()
	_, ctx := createContext(ts)
	clientID := "sa-" + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), 12)
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddClient(ctx, &schemas.Client{
		ClientID:                clientID,
		Kind:                    constants.ClientKindServiceAccount,
		Name:                    "test-sa",
		ClientSecret:            string(hash),
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodClientSecretPost,
		IsActive:                true,
	})
	require.NoError(t, err)
	return clientID
}

// TestClientCheckMiddleware_RegistryAware verifies the X-Authorizer-Client-ID
// middleware now resolves against the client registry, while preserving the
// backward-compat allowances: an empty header passes (SDKs that don't send it),
// and the reserved client_id passes. A bogus id is rejected; a registered client
// id is accepted.
func TestClientCheckMiddleware_RegistryAware(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registeredClientID := addServiceAccountClient(t, ts, "sa-secret")

	router := gin.New()
	router.Use(ts.HttpProvider.ClientCheckMiddleware())
	router.POST("/graphql", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	call := func(clientID string, sendHeader bool) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/graphql", nil)
		if sendHeader {
			req.Header.Set("X-Authorizer-Client-ID", clientID)
		}
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("empty_header_allowed_backward_compat", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, call("", false).Code)
	})
	t.Run("explicit_empty_header_allowed", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, call("", true).Code)
	})
	t.Run("reserved_client_id_allowed", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, call(cfg.ClientID, true).Code)
	})
	t.Run("registered_client_id_allowed", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, call(registeredClientID, true).Code)
	})
	t.Run("bogus_client_id_rejected", func(t *testing.T) {
		w := call("definitely-not-a-client", true)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "invalid_client_id", body["error"])
	})
}

// TestIntrospect_RegistryAwareCallerAndAudience verifies that a registered
// (non-reserved) client authenticates as the introspection caller through the
// registry (RFC 7662 §2.1), and that the audience check is registry-aware: a
// token minted for a different client (aud != caller's client_id) returns
// {"active": false} — no oracle, no claim leakage across clients.
func TestIntrospect_RegistryAwareCallerAndAudience(t *testing.T) {
	// Token whose aud is the reserved client (Config.ClientID).
	ts, _, authToken := setupIntrospectTest(t)
	cfg := ts.Config
	saSecret := "sa-introspect-secret"
	saClientID := addServiceAccountClient(t, ts, saSecret)

	// The service_account caller authenticates successfully (200), but the token's
	// aud is the reserved client, not the caller — so it is reported inactive.
	form := url.Values{}
	form.Set("token", authToken.AccessToken.Token)
	form.Set("client_id", saClientID)
	form.Set("client_secret", saSecret)
	w := postIntrospect(t, ts, form.Encode())
	require.Equal(t, http.StatusOK, w.Code, "authenticated caller must not be rejected")
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["active"], "token minted for another client MUST be inactive for this caller")
	assert.Nil(t, body["sub"], "inactive response MUST NOT leak sub across clients")
	assert.Nil(t, body["client_id"], "inactive response MUST NOT leak client_id")

	// Sanity: the reserved caller (via Config fallback) still sees it active — the
	// audience check keys on the caller, and here caller == token aud.
	form2 := url.Values{}
	form2.Set("token", authToken.AccessToken.Token)
	form2.Set("client_id", cfg.ClientID)
	form2.Set("client_secret", cfg.ClientSecret)
	w2 := postIntrospect(t, ts, form2.Encode())
	require.Equal(t, http.StatusOK, w2.Code)
	var body2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body2))
	assert.Equal(t, true, body2["active"])
	assert.Equal(t, cfg.ClientID, body2["client_id"])
}

// TestIntrospect_UnauthenticatedCallerRejected verifies RFC 7662 §2.1: a caller
// that presents no client credential is rejected, not answered with active:false.
func TestIntrospect_UnauthenticatedCallerRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	form := url.Values{}
	form.Set("token", "some-token")
	// No client_id / client_secret at all.
	w := postIntrospect(t, ts, form.Encode())
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_request", body["error"], "missing client_id MUST be rejected per RFC 7662 §2.1")
}

// TestRevoke_ForeignClientCannotRevoke verifies RFC 7009 token-ownership: a
// different authenticated client cannot revoke a token issued to the reserved
// client. The response is 200 (no oracle) but the session is left intact; the
// legitimate owner then revokes it successfully.
func TestRevoke_ForeignClientCannotRevoke(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "revoke_own_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Scope:    []string{"openid", "email", "profile", "offline_access"},
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.RefreshToken)
	refreshToken := *loginRes.RefreshToken

	// Compute the session key the handler uses, to assert the session's presence.
	claims, err := ts.TokenProvider.ParseJWTToken(refreshToken)
	require.NoError(t, err)
	userID := claims["sub"].(string)
	nonce := claims["nonce"].(string)
	sessionToken := userID
	if lm, ok := claims["login_method"].(string); ok && lm != "" {
		sessionToken = lm + ":" + userID
	}
	sessionKey := constants.TokenTypeRefreshToken + "_" + nonce

	saClientID := addServiceAccountClient(t, ts, "sa-revoke-secret")

	router := gin.New()
	router.POST("/oauth/revoke", ts.HttpProvider.RevokeRefreshTokenHandler())

	doRevoke := func(clientID string) *httptest.ResponseRecorder {
		form := url.Values{}
		form.Set("token", refreshToken)
		form.Set("client_id", clientID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)
		return w
	}

	// Foreign client: 200 (no oracle) but MUST NOT revoke.
	w := doRevoke(saClientID)
	assert.Equal(t, http.StatusOK, w.Code)
	got, _ := ts.MemoryStoreProvider.GetUserSession(sessionToken, sessionKey)
	assert.NotEmpty(t, got, "foreign client MUST NOT revoke another client's token")

	// Legitimate owner (reserved client): revokes successfully.
	w = doRevoke(cfg.ClientID)
	assert.Equal(t, http.StatusOK, w.Code)
	got, _ = ts.MemoryStoreProvider.GetUserSession(sessionToken, sessionKey)
	assert.Empty(t, got, "token owner MUST be able to revoke its token")
}

// TestDiscovery_AdvertisesNewCapabilities verifies the discovery metadata now
// advertises client_credentials (grant) and private_key_jwt (token-endpoint auth
// method) per §4.6 / RFC 8414 / OIDC Discovery.
func TestDiscovery_AdvertisesNewCapabilities(t *testing.T) {
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

	assert.Contains(t, toStringSlice(body["grant_types_supported"]), constants.GrantTypeClientCredentials,
		"grant_types_supported MUST advertise client_credentials")
	assert.Contains(t, toStringSlice(body["token_endpoint_auth_methods_supported"]), "private_key_jwt",
		"token_endpoint_auth_methods_supported MUST advertise private_key_jwt")
}

func toStringSlice(v interface{}) []string {
	arr, _ := v.([]interface{})
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
