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

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// TestHybridFlowCodeExchange exercises the full OIDC hybrid flow:
// /authorize (with session cookie) -> extract code -> /oauth/token exchange.
// This caught a bug where the hybrid path stored the raw nonce (FingerPrint)
// instead of the AES-encrypted session (FingerPrintHash), causing
// ValidateBrowserSession to fail at token exchange time.
func TestHybridFlowCodeExchange(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// 1. Create a test user
	email := "hybrid_exchange_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	// 2. Create a session token for the user so we can set the cookie
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	nonce := uuid.New().String()
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       cfg.DefaultRoles,
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)
	require.NotEmpty(t, sessionToken)

	// Store the session in the memory store so ValidateBrowserSession can find it
	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	err = ts.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+sessionData.Nonce, sessionToken, sessionExpiresAt)
	require.NoError(t, err)

	// 3. Call /authorize with hybrid response_type and a valid session cookie
	codeVerifier := uuid.New().String() + uuid.New().String()
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	state := uuid.New().String()
	authNonce := uuid.New().String()
	qs := url.Values{}
	qs.Set("client_id", cfg.ClientID)
	qs.Set("response_type", "code id_token")
	qs.Set("redirect_uri", "http://localhost:3000/callback")
	qs.Set("code_challenge", codeChallenge)
	qs.Set("code_challenge_method", "S256")
	qs.Set("state", state)
	qs.Set("nonce", authNonce)
	qs.Set("response_mode", "fragment")
	qs.Set("scope", "openid profile email")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/authorize?"+qs.Encode(), nil)
	// Set session cookie so the authorize handler finds a valid session
	req.AddCookie(&http.Cookie{
		Name:  constants.AppCookieName + "_session",
		Value: sessionToken,
	})
	router.ServeHTTP(w, req)

	// 4. Extract the code from the fragment redirect
	require.Equal(t, http.StatusFound, w.Code, "authorize should redirect with 302")
	location := w.Header().Get("Location")
	require.NotEmpty(t, location, "Location header must be present")

	// Parse the fragment from the redirect URI
	parsedURL, err := url.Parse(location)
	require.NoError(t, err)
	fragment := parsedURL.Fragment
	require.NotEmpty(t, fragment, "fragment must contain authorization response params")

	fragmentValues, err := url.ParseQuery(fragment)
	require.NoError(t, err)

	code := fragmentValues.Get("code")
	require.NotEmpty(t, code, "code must be present in hybrid response")
	assert.NotEmpty(t, fragmentValues.Get("id_token"), "id_token must be present in code+id_token hybrid response")
	assert.Equal(t, state, fragmentValues.Get("state"), "state must be echoed back")

	// 5. Exchange the code at /oauth/token
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("redirect_uri", "http://localhost:3000/callback")

	tokenW := httptest.NewRecorder()
	tokenReq, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(tokenW, tokenReq)

	// 6. Assert the token exchange succeeds
	var tokenBody map[string]interface{}
	require.NoError(t, json.Unmarshal(tokenW.Body.Bytes(), &tokenBody))
	assert.Equal(t, http.StatusOK, tokenW.Code,
		"token exchange must succeed; got error=%v error_description=%v",
		tokenBody["error"], tokenBody["error_description"])
	assert.NotEmpty(t, tokenBody["access_token"], "access_token must be present")
	assert.NotEmpty(t, tokenBody["id_token"], "id_token must be present")
}
