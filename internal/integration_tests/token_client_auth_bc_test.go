package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/authorizerdev/authorizer/internal/token"
)

// runReservedLogin performs the interactive reserved-client login up to the
// authorization code: signup -> browser session -> /authorize (PKCE, S256). It
// returns a router with /oauth/token wired, the single-use code, and the PKCE
// code_verifier so each test can drive the token exchange under different client
// authentication conditions. This is the exact flow the reserved client uses in
// production, so a green exchange proves the client-auth resolver preserves BC1.
func runReservedLogin(t *testing.T, ts *testSetup, clientID string, defaultRoles []string) (http.Handler, string, string) {
	t.Helper()
	_, ctx := createContext(ts)

	email := "reserved_bc_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	nonce := uuid.New().String()
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       defaultRoles,
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey, constants.TokenTypeSessionToken+"_"+sessionData.Nonce, sessionToken, sessionExpiresAt))

	codeVerifier := uuid.New().String() + uuid.New().String()
	sum := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(sum[:])

	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	qs := url.Values{}
	qs.Set("client_id", clientID)
	qs.Set("response_type", "code")
	qs.Set("redirect_uri", "http://localhost:3000/callback")
	qs.Set("code_challenge", codeChallenge)
	qs.Set("code_challenge_method", "S256")
	qs.Set("state", uuid.New().String())
	qs.Set("response_mode", "query")
	qs.Set("scope", "openid profile email")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/authorize?"+qs.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code, "authorize should redirect: %s", w.Body.String())
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code, "authorization code must be present")

	return router, code, codeVerifier
}

func exchangeCode(router http.Handler, form url.Values, basicAuth []string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basicAuth != nil {
		req.SetBasicAuth(basicAuth[0], basicAuth[1])
	}
	router.ServeHTTP(w, req)
	return w
}

// BC1 (fallback): with NO reserved-client row seeded (the read-only-replica
// case), the reserved client still authenticates the authorization_code+PKCE
// exchange against Config.ClientID / Config.ClientSecret. This is the default
// state of a fresh test DB, so it also guards every other login test.
func TestReservedClientLogin_ConfigFallback_SeedAbsent(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	// Guard: no reserved-client row exists — we are genuinely on the fallback path.
	existing, err := ts.StorageProvider.GetClientByClientID(t.Context(), cfg.ClientID)
	require.True(t, err != nil || existing == nil, "test must run with the reserved row absent")

	router, code, codeVerifier := runReservedLogin(t, ts, cfg.ClientID, cfg.DefaultRoles)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("redirect_uri", "http://localhost:3000/callback")

	w := exchangeCode(router, form, nil)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body["access_token"])
}

// BC1 (registry): with the reserved-client row seeded (bcrypt(Config.ClientSecret),
// client_id == Config.ClientID), the same login exchanges successfully through
// the resolver's registry path — proving the row-backed path is byte-for-byte
// equivalent to the fallback for the reserved client.
func TestReservedClientLogin_SeededRow(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.ClientSecret), 12)
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddClient(t.Context(), &schemas.Client{
		ClientID:                cfg.ClientID,
		Kind:                    constants.ClientKindInteractive,
		Name:                    "Reserved Interactive Client",
		ClientSecret:            string(hash),
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodClientSecretBasic,
		IsActive:                true,
	})
	require.NoError(t, err)

	router, code, codeVerifier := runReservedLogin(t, ts, cfg.ClientID, cfg.DefaultRoles)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("redirect_uri", "http://localhost:3000/callback")

	w := exchangeCode(router, form, nil)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body["access_token"])
}

// RFC 6749 §2.3: presenting more than one client-authentication method (HTTP
// Basic AND a body client_secret) must be rejected with invalid_request, before
// any grant processing. New behavior enforced by the resolver.
func TestTokenClientAuth_DualMethodRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret) // client_secret_post
	form.Set("code", "any-code")
	form.Set("code_verifier", "any-verifier")

	// ...and ALSO HTTP Basic (client_secret_basic) in the same request.
	w := exchangeCode(router, form, []string{cfg.ClientID, cfg.ClientSecret})
	require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_request", body["error"],
		"RFC 6749 §2.3: more than one auth method MUST be rejected as invalid_request")
}
