package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// issueAccessTokenWithScopes mints an access token with the requested scope
// set so we can hit /userinfo as an authenticated caller without spinning
// up the full /authorize → /token dance. It also persists the access-token
// nonce in the memory store, mirroring what the real login flow does, so
// that ValidateAccessToken accepts the token.
func issueAccessTokenWithScopes(t *testing.T, ts *testSetup, ctx context.Context, email string, scopes []string) string {
	t.Helper()
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	nonce := "nonce-" + uuid.New().String()
	authToken, err := ts.TokenProvider.CreateAuthToken(nil, &token.AuthTokenConfig{
		User:        user,
		Roles:       []string{"user"},
		Scope:       scopes,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		Nonce:       nonce,
		HostName:    "http://localhost",
	})
	require.NoError(t, err)
	require.NotNil(t, authToken.AccessToken)

	// Mirror the real login flow: persist session + access token in the
	// memory store so ValidateAccessToken's nonce lookup succeeds.
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionStoreKey,
		constants.TokenTypeSessionToken+"_"+authToken.FingerPrint,
		authToken.FingerPrintHash,
		authToken.SessionTokenExpiresAt,
	))
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionStoreKey,
		constants.TokenTypeAccessToken+"_"+authToken.FingerPrint,
		authToken.AccessToken.Token,
		authToken.AccessToken.ExpiresAt,
	))

	return authToken.AccessToken.Token
}

func callUserInfo(t *testing.T, ts *testSetup, accessToken string) (int, map[string]interface{}) {
	t.Helper()
	router := gin.New()
	router.GET("/userinfo", ts.HttpProvider.UserInfoHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)
	var body map[string]interface{}
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &body)
	}
	return w.Code, body
}

func signupForUserInfoTests(t *testing.T, ts *testSetup, ctx context.Context) string {
	t.Helper()
	email := "userinfo_phase1_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	return email
}

// TestUserInfoScopeFiltering covers OIDC Core §5.4 scope-based claim
// filtering on /userinfo. Filtering is unconditional — there is no opt-in
// flag. The returned response always contains `sub` plus only the claims
// permitted by the standard scope groups encoded in the access token.
func TestUserInfoScopeFiltering(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := signupForUserInfoTests(t, ts, ctx)

	t.Run("only_openid_returns_only_sub", func(t *testing.T) {
		at := issueAccessTokenWithScopes(t, ts, ctx, email, []string{"openid"})
		code, body := callUserInfo(t, ts, at)
		require.Equal(t, http.StatusOK, code)

		assert.NotEmpty(t, body["sub"], "sub MUST always be present")
		assert.Nil(t, body["email"], "email scope not requested → email must be omitted")
		assert.Nil(t, body["email_verified"], "email scope not requested → email_verified must be omitted")
		assert.Nil(t, body["given_name"], "profile scope not requested → given_name must be omitted")
		assert.Nil(t, body["family_name"], "profile scope not requested → family_name must be omitted")
	})

	t.Run("openid_email_returns_email_claims", func(t *testing.T) {
		at := issueAccessTokenWithScopes(t, ts, ctx, email, []string{"openid", "email"})
		code, body := callUserInfo(t, ts, at)
		require.Equal(t, http.StatusOK, code)

		assert.NotEmpty(t, body["sub"])
		assert.Equal(t, email, body["email"])
		_, hasVerified := body["email_verified"]
		assert.True(t, hasVerified, "email_verified must be returned with the email scope")
		assert.Nil(t, body["given_name"])
	})

	t.Run("openid_profile_returns_profile_claims_no_email", func(t *testing.T) {
		at := issueAccessTokenWithScopes(t, ts, ctx, email, []string{"openid", "profile"})
		code, body := callUserInfo(t, ts, at)
		require.Equal(t, http.StatusOK, code)

		assert.NotEmpty(t, body["sub"])
		// Profile claims may legitimately be empty/nil values on a freshly
		// signed-up user, but the KEYS must be present in the response.
		_, hasGiven := body["given_name"]
		_, hasFamily := body["family_name"]
		_, hasNickname := body["nickname"]
		_, hasPreferred := body["preferred_username"]
		assert.True(t, hasGiven, "profile scope → given_name key present")
		assert.True(t, hasFamily, "profile scope → family_name key present")
		assert.True(t, hasNickname, "profile scope → nickname key present")
		assert.True(t, hasPreferred, "profile scope → preferred_username key present")
		assert.Nil(t, body["email"], "email scope not requested → email omitted")
	})

	t.Run("openid_profile_email_returns_both_groups", func(t *testing.T) {
		at := issueAccessTokenWithScopes(t, ts, ctx, email, []string{"openid", "profile", "email"})
		code, body := callUserInfo(t, ts, at)
		require.Equal(t, http.StatusOK, code)

		assert.NotEmpty(t, body["sub"])
		assert.Equal(t, email, body["email"])
		_, hasGiven := body["given_name"]
		assert.True(t, hasGiven)
	})

	t.Run("sub_is_always_present_for_unknown_scope_combo", func(t *testing.T) {
		// Some random custom scope that doesn't map to any standard claim group.
		at := issueAccessTokenWithScopes(t, ts, ctx, email, []string{"openid", "custom:thing"})
		code, body := callUserInfo(t, ts, at)
		require.Equal(t, http.StatusOK, code)
		assert.NotEmpty(t, body["sub"])
		assert.Nil(t, body["email"])
		assert.Nil(t, body["given_name"])
	})
}
