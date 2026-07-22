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

// loginForOfflineAccess is runReservedLogin's counterpart with an
// offline_access scope, so the resulting code exchange mints a refresh
// token — needed to exercise refresh-token client binding, which
// runReservedLogin's fixed "openid profile email" scope never triggers.
func loginForOfflineAccess(t *testing.T, ts *testSetup, clientID string) (http.Handler, string, string) {
	t.Helper()
	_, ctx := createContext(ts)

	email := "refresh_bind_" + uuid.New().String() + "@authorizer.dev"
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
	scope := []string{"openid", "profile", "email", "offline_access"}
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       ts.Config.DefaultRoles,
		Scope:       scope,
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
	qs.Set("scope", strings.Join(scope, " "))

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

func registerTestClient(t *testing.T, ts *testSetup, clientID, secret string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), 12)
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddClient(t.Context(), &schemas.Client{
		ClientID:                clientID,
		Kind:                    constants.ClientKindInteractive,
		Name:                    "Test Client " + clientID,
		ClientSecret:            string(hash),
		TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodClientSecretBasic,
		IsActive:                true,
	})
	require.NoError(t, err)
}

// RFC 6749 §6: "the authorization server MUST verify the binding between the
// refresh token and client identity whenever possible." A refresh token
// minted for client1 must not be redeemable by client2 presenting its own
// valid credentials — regression test for the oidcc-refresh-token
// conformance failure (LOG-0234: ValidateErrorFromTokenEndpointResponseError
// — the server returned no error when a refresh token issued to one client
// was redeemed by a different client).
func TestRefreshToken_CrossClientRedemption_Rejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	registerTestClient(t, ts, "client-one", "client-one-secret")
	registerTestClient(t, ts, "client-two", "client-two-secret")

	router, code, codeVerifier := loginForOfflineAccess(t, ts, "client-one")

	exchangeForm := url.Values{}
	exchangeForm.Set("grant_type", "authorization_code")
	exchangeForm.Set("code", code)
	exchangeForm.Set("code_verifier", codeVerifier)
	exchangeForm.Set("redirect_uri", "http://localhost:3000/callback")

	w := exchangeCode(router, exchangeForm, []string{"client-one", "client-one-secret"})
	require.Equal(t, http.StatusOK, w.Code, "code exchange body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	refreshToken, _ := body["refresh_token"].(string)
	require.NotEmpty(t, refreshToken, "offline_access scope must yield a refresh_token")

	t.Run("a different client redeeming the refresh token is rejected", func(t *testing.T) {
		refreshForm := url.Values{}
		refreshForm.Set("grant_type", "refresh_token")
		refreshForm.Set("refresh_token", refreshToken)

		w := exchangeCode(router, refreshForm, []string{"client-two", "client-two-secret"})
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		var errBody map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
		assert.Equal(t, "invalid_grant", errBody["error"])
	})

	t.Run("the issuing client can still redeem its own refresh token", func(t *testing.T) {
		refreshForm := url.Values{}
		refreshForm.Set("grant_type", "refresh_token")
		refreshForm.Set("refresh_token", refreshToken)

		w := exchangeCode(router, refreshForm, []string{"client-one", "client-one-secret"})
		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		var okBody map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &okBody))
		assert.NotEmpty(t, okBody["access_token"])
	})
}

// TestRefreshToken_WrongSecretPresented_Rejected is the regression test for
// the refresh-token secret-verification hardening finding: a confidential
// client presenting an incorrect secret on refresh_token must be rejected
// instead of silently authenticated on client_id alone, while a public
// client presenting no secret at all is unaffected.
func TestRefreshToken_WrongSecretPresented_Rejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	registerTestClient(t, ts, "client-secret-check", "correct-secret")

	router, code, codeVerifier := loginForOfflineAccess(t, ts, "client-secret-check")

	exchangeForm := url.Values{}
	exchangeForm.Set("grant_type", "authorization_code")
	exchangeForm.Set("code", code)
	exchangeForm.Set("code_verifier", codeVerifier)
	exchangeForm.Set("redirect_uri", "http://localhost:3000/callback")

	w := exchangeCode(router, exchangeForm, []string{"client-secret-check", "correct-secret"})
	require.Equal(t, http.StatusOK, w.Code, "code exchange body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	refreshToken, _ := body["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)

	t.Run("a presented wrong secret is rejected even though client_id matches", func(t *testing.T) {
		refreshForm := url.Values{}
		refreshForm.Set("grant_type", "refresh_token")
		refreshForm.Set("refresh_token", refreshToken)

		w := exchangeCode(router, refreshForm, []string{"client-secret-check", "definitely-wrong-secret"})
		// 401, not 400: exchangeCode presents credentials via HTTP Basic
		// (req.SetBasicAuth), and respondClientAuthError maps ErrInvalidClient
		// to 401 (with WWW-Authenticate) whenever hasBasicAuth is true.
		assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	})

	t.Run("no secret presented (public client shape) still succeeds", func(t *testing.T) {
		refreshForm := url.Values{}
		refreshForm.Set("grant_type", "refresh_token")
		refreshForm.Set("refresh_token", refreshToken)
		refreshForm.Set("client_id", "client-secret-check")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(refreshForm.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})
}
