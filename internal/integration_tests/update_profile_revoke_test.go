package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestUpdateProfilePasswordOnlyChangeRevokesSessions pins that a password-only
// change through UpdateProfile terminates every pre-existing session and
// refresh token, exactly as reset_password does.
//
// It did not (GHSA-qx8r-9p3g-vcp4). DeleteAllUserSessions lived only inside the
// email-change branch, so the "change my password" flow — the single
// containment action a compromised user takes — left every previously issued
// refresh token live for its full TTL (30 days by default). The victim was told
// the change succeeded while the attacker's credential kept working.
//
// The revocation is asserted through the token validator rather than by reading
// the store, because the store holds a digest and because the validator is what
// an attacker's replay would actually go through.
func TestUpdateProfilePasswordOnlyChangeRevokesSessions(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "update_profile_revoke_" + uuid.New().String() + "@authorizer.dev"
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
	require.NotNil(t, loginRes.AccessToken)
	require.NotNil(t, loginRes.RefreshToken, "offline_access must yield a refresh token")

	stolenAccess := *loginRes.AccessToken
	stolenRefresh := *loginRes.RefreshToken
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+stolenAccess)

	// Control: both credentials are live before the password change. Without
	// this the test could pass for the wrong reason (a token that was never
	// valid).
	res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     stolenAccess,
		TokenType: constants.TokenTypeAccessToken,
	})
	require.NoError(t, err)
	require.True(t, res.IsValid, "control: the access token must be live before the change")

	res, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     stolenRefresh,
		TokenType: constants.TokenTypeRefreshToken,
	})
	require.NoError(t, err)
	require.True(t, res.IsValid, "control: the refresh token must be live before the change")

	newPassword := "NewPassword@456"
	_, err = ts.GraphQLProvider.UpdateProfile(ctx, &model.UpdateProfileRequest{
		OldPassword:        refs.NewStringRef(password),
		NewPassword:        refs.NewStringRef(newPassword),
		ConfirmNewPassword: refs.NewStringRef(newPassword),
	})
	require.NoError(t, err)

	// The attack: replay the pre-change credentials.
	_, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     stolenAccess,
		TokenType: constants.TokenTypeAccessToken,
	})
	require.Error(t, err, "a password change must revoke pre-existing access tokens")

	_, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     stolenRefresh,
		TokenType: constants.TokenTypeRefreshToken,
	})
	require.Error(t, err, "a password change must revoke pre-existing refresh tokens")
}

// TestUpdateProfileEmailChangeStillRevokes guards the branch that was already
// correct. The fix hoists the revocation out of the email-change branch, and the
// flag it is now gated on must not be the pre-existing hasEmailChanged — that
// one is set only when email verification is enabled, while the revocation
// fires for every email change. Reusing it would silently drop revocation here.
func TestUpdateProfileEmailChangeStillRevokes(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableEmailVerification = false
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "update_profile_email_revoke_" + uuid.New().String() + "@authorizer.dev"
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
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.AccessToken)
	stolenAccess := *loginRes.AccessToken
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+stolenAccess)

	newEmail := "update_profile_email_revoke_new_" + uuid.New().String() + "@authorizer.dev"
	_, err = ts.GraphQLProvider.UpdateProfile(ctx, &model.UpdateProfileRequest{
		Email: refs.NewStringRef(newEmail),
	})
	require.NoError(t, err)

	_, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     stolenAccess,
		TokenType: constants.TokenTypeAccessToken,
	})
	require.Error(t, err, "an email change must still revoke pre-existing sessions")
}

// TestUpdateProfileNonCredentialChangeKeepsSession pins the other side of the
// contract: editing a display name is not a security event and must NOT log the
// caller out. Without this, "revoke on every update" would look like a passing
// fix while breaking every profile-edit screen.
func TestUpdateProfileNonCredentialChangeKeepsSession(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "update_profile_keep_" + uuid.New().String() + "@authorizer.dev"
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
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.AccessToken)
	accessToken := *loginRes.AccessToken
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+accessToken)

	_, err = ts.GraphQLProvider.UpdateProfile(ctx, &model.UpdateProfileRequest{
		GivenName: refs.NewStringRef("Renamed"),
	})
	require.NoError(t, err)

	res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     accessToken,
		TokenType: constants.TokenTypeAccessToken,
	})
	require.NoError(t, err, "a display-name edit must not revoke the caller's session")
	require.True(t, res.IsValid)
}
