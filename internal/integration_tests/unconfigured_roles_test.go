package integration_tests

import (
	"fmt"
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

// claimStrings returns a JWT claim that is a JSON array of strings.
func claimStrings(t *testing.T, claims map[string]interface{}, key string) []string {
	t.Helper()
	raw, ok := claims[key].([]interface{})
	require.True(t, ok, "claim %q missing or not an array: %#v", key, claims[key])
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		require.True(t, ok)
		out = append(out, s)
	}
	return out
}

// TestRoleRemovedFromConfigIsNotHonoured covers issue #425: when a role is
// removed from --roles, users who already hold it must stop being granted it.
// The stored user record is left intact, so re-adding the role restores it.
func TestRoleRemovedFromConfigIsNotHonoured(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "unconfigured_roles_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
		Roles:           []string{"user", "editor"},
	})
	require.NoError(t, err)

	// A session and access token minted while "editor" is still configured.
	oldLogin, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Roles:    []string{"user", "editor"},
	})
	require.NoError(t, err)
	require.NotNil(t, oldLogin.AccessToken)
	oldSession := latestAppSessionCookie(ts)
	require.NotEmpty(t, oldSession)
	oldAccessToken := *oldLogin.AccessToken

	// Simulate the operator restarting with "editor" removed from --roles.
	configuredRoles := cfg.Roles
	withoutEditor := []string{}
	for _, r := range configuredRoles {
		if r != "editor" {
			withoutEditor = append(withoutEditor, r)
		}
	}
	cfg.Roles = withoutEditor
	t.Cleanup(func() { cfg.Roles = configuredRoles })

	t.Run("login requesting the removed role fails", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email,
			Password: password,
			Roles:    []string{"editor"},
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("tokens do not carry the removed role", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email,
			Password: password,
		})
		require.NoError(t, err)
		require.NotNil(t, res.AccessToken)
		require.NotNil(t, res.IDToken)

		at, err := ts.TokenProvider.ParseJWTToken(*res.AccessToken)
		require.NoError(t, err)
		assert.NotContains(t, claimStrings(t, at, "allowed_roles"), "editor")
		assert.Contains(t, claimStrings(t, at, "allowed_roles"), "user")

		idt, err := ts.TokenProvider.ParseJWTToken(*res.IDToken)
		require.NoError(t, err)
		assert.NotContains(t, claimStrings(t, idt, "allowed_roles"), "editor")
		assert.NotContains(t, claimStrings(t, idt, cfg.JWTRoleClaim), "editor")
	})

	t.Run("a pre-existing session no longer satisfies the removed role", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
			Cookie: oldSession,
			Roles:  []string{"editor"},
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
			Cookie: oldSession,
			Roles:  []string{"user"},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.IsValid)
	})

	t.Run("a pre-existing access token no longer satisfies the removed role", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			TokenType: "access_token",
			Token:     oldAccessToken,
			Roles:     []string{"editor"},
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			TokenType: "access_token",
			Token:     oldAccessToken,
			Roles:     []string{"user"},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.IsValid)
	})

	t.Run("re-issued tokens drop the removed role but keep roles not sourced from the user record", func(t *testing.T) {
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		// "org_admin" stands in for an FGA org-group-derived role: it is not
		// configured and not on the user record, so it must survive.
		authToken, err := ts.TokenProvider.CreateAuthToken(ts.GinContext, &token.AuthTokenConfig{
			User:        user,
			Roles:       []string{"user", "editor", "org_admin"},
			Scope:       []string{"openid", "offline_access"},
			LoginMethod: "basic_auth",
			Nonce:       uuid.NewString(),
			HostName:    testAuthorizerHost(ts),
		})
		require.NoError(t, err)

		for name, tok := range map[string]string{
			"access":  authToken.AccessToken.Token,
			"refresh": authToken.RefreshToken.Token,
		} {
			claims, err := ts.TokenProvider.ParseJWTToken(tok)
			require.NoError(t, err, name)
			assert.Equal(t, []string{"user", "org_admin"}, claimStrings(t, claims, "roles"), name)
			assert.Equal(t, []string{"user"}, claimStrings(t, claims, "allowed_roles"), name)
		}
		idt, err := ts.TokenProvider.ParseJWTToken(authToken.IDToken.Token)
		require.NoError(t, err)
		assert.Equal(t, []string{"user", "org_admin"}, claimStrings(t, idt, cfg.JWTRoleClaim))
	})

	t.Run("stored roles are left intact so re-adding the role restores it", func(t *testing.T) {
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Contains(t, user.Roles, "editor")

		cfg.Roles = configuredRoles
		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email,
			Password: password,
			Roles:    []string{"editor"},
		})
		require.NoError(t, err)
		require.NotNil(t, res.AccessToken)
		at, err := ts.TokenProvider.ParseJWTToken(*res.AccessToken)
		require.NoError(t, err)
		assert.Contains(t, claimStrings(t, at, "allowed_roles"), "editor")
		cfg.Roles = withoutEditor
	})
}

// TestRoleRemovedFromConfigOnReissue covers the paths that re-issue tokens from
// a credential minted before the role was removed: the refresh grant at
// /oauth/token and the session query (#425).
func TestRoleRemovedFromConfigOnReissue(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "unconfigured_roles_reissue_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
		Roles:           []string{"user", "editor"},
	})
	require.NoError(t, err)
	oldLogin, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Roles:    []string{"user", "editor"},
		Scope:    []string{"openid", "email", "profile", "offline_access"},
	})
	require.NoError(t, err)
	require.NotNil(t, oldLogin.RefreshToken)
	// A separate login for the session query: rotating the refresh token
	// retires its session, so the two credentials must not share a nonce.
	_, err = ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Roles:    []string{"user", "editor"},
	})
	require.NoError(t, err)
	oldSession := latestAppSessionCookie(ts)
	require.NotEmpty(t, oldSession)

	configuredRoles := cfg.Roles
	cfg.Roles = []string{"user", "admin"}
	t.Cleanup(func() { cfg.Roles = configuredRoles })

	t.Run("refresh grant with an old refresh token drops the removed role", func(t *testing.T) {
		router := gin.New()
		router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", *oldLogin.RefreshToken)
		form.Set("client_id", cfg.ClientID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		body := decodeJSON(t, w)

		assert.Equal(t, []string{"user"}, claimStrings(t, body, "roles"), "response roles")
		for _, key := range []string{"access_token", "refresh_token"} {
			tok, _ := body[key].(string)
			require.NotEmpty(t, tok, key)
			claims, err := ts.TokenProvider.ParseJWTToken(tok)
			require.NoError(t, err, key)
			assert.Equal(t, []string{"user"}, claimStrings(t, claims, "roles"), key)
			assert.Equal(t, []string{"user"}, claimStrings(t, claims, "allowed_roles"), key)
		}
	})

	t.Run("session query with an old session drops the removed role", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, oldSession))
		_, err := ts.GraphQLProvider.Session(ctx, &model.SessionQueryRequest{Roles: []string{"editor"}})
		assert.Error(t, err)

		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, oldSession))
		res, err := ts.GraphQLProvider.Session(ctx, &model.SessionQueryRequest{})
		require.NoError(t, err)
		require.NotNil(t, res.AccessToken)
		claims, err := ts.TokenProvider.ParseJWTToken(*res.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, []string{"user"}, claimStrings(t, claims, "roles"))
		assert.Equal(t, []string{"user"}, claimStrings(t, claims, "allowed_roles"))
	})
}

// TestVerifyEmailRejectsRemovedRole covers /verify_email?roles=: a role the
// user holds but that is no longer configured cannot be requested (#425).
func TestVerifyEmailRejectsRemovedRole(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "unconfigured_roles_verify_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
		Roles:           []string{"user", "editor"},
	})
	require.NoError(t, err)
	vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	configuredRoles := cfg.Roles
	cfg.Roles = []string{"user", "admin"}
	t.Cleanup(func() { cfg.Roles = configuredRoles })

	resp, err := noRedirectClient().Get(
		testAuthorizerHost(ts) + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
			"&roles=editor&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	for _, c := range resp.Cookies() {
		if strings.Contains(c.Name, "session") {
			assert.Empty(t, c.Value, "no session may be issued for a removed role")
		}
	}
	if resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusFound {
		// The rejection must be the role check, not some other failure.
		assert.Contains(t, resp.Header.Get("Location"), "error=invalid roles")
	} else {
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	}
}
