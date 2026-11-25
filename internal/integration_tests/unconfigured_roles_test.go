package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
