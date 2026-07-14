package integration_tests

import (
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// registerPasskeyForNewUser signs up a fresh verified user, registers one
// passkey via a simulated ceremony, and returns everything a login-time
// assertion needs. Mirrors the setup in webauthn_test.go's
// TestWebauthnPasskeyRegistrationAndLogin.
func registerPasskeyForNewUser(t *testing.T, ts *testSetup) (*schemas.User, virtualwebauthn.RelyingParty, virtualwebauthn.Authenticator, virtualwebauthn.Credential) {
	t.Helper()
	req, ctx := createContext(ts)
	rp := testRelyingParty(t, ts)
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	email := "enforce_mfa_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes.AccessToken)
	req.Header.Set("Authorization", "Bearer "+*signupRes.AccessToken)

	optRes, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil)
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(optRes.Options)
	require.NoError(t, err)
	authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
		UserHandle: []byte(attOpts.UserID),
	})
	authenticator.AddCredential(credential)
	attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
	_, err = ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{Credential: attResp})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	req.Header.Del("Authorization")
	return user, rp, authenticator, credential
}

func assertPasskeyLogin(t *testing.T, ts *testSetup, rp virtualwebauthn.RelyingParty, authenticator virtualwebauthn.Authenticator, credential virtualwebauthn.Credential) (*model.AuthResponse, error) {
	t.Helper()
	_, ctx := createContext(ts)
	optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, nil)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
	require.NoError(t, err)
	assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
	return ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
}

func TestWebauthnLoginVerifyEnforceMFA(t *testing.T) {
	t.Run("EnforceMFA=false — unchanged, issues token", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes.AccessToken, "EnforceMFA=false must not block passkey login")
	})

	t.Run("EnforceMFA=true, user MFA not individually enabled — unaffected", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		// signup.go force-enables IsMultiFactorAuthEnabled for every new user
		// while EnforceMFA is on, so a freshly signed-up user can't land here
		// with it unset. Explicitly turn it back off — mirrors an admin
		// disabling MFA for one account (admin_users.go allows this even
		// under EnforceMFA) — to exercise the gate's own precondition.
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(false)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes.AccessToken)
	})

	t.Run("EnforceMFA=true, TOTP verified — blocks token, offers totp screen", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)
		now := time.Now().Unix()
		_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
			UserID: user.ID, Method: constants.EnvKeyTOTPAuthenticator,
			Secret: "dummy-secret", VerifiedAt: &now,
		})
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		assert.Nil(t, authRes.AccessToken, "a user with verified TOTP must not get a token straight off a passkey login when MFA is enforced")
		assert.True(t, refs.BoolValue(authRes.ShouldShowTotpScreen))
		assert.Nil(t, authRes.AuthenticatorSecret, "already-enrolled path must not hand back a fresh enrollment payload")
	})

	t.Run("EnforceMFA=true, TOTP not enrolled — blocks token, returns enrollment payload", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		assert.Nil(t, authRes.AccessToken)
		assert.True(t, refs.BoolValue(authRes.ShouldShowTotpScreen))
		assert.NotNil(t, authRes.AuthenticatorSecret, "not-yet-enrolled path must hand back a fresh TOTP enrollment payload")
	})

	t.Run("EnforceMFA=true, TOTP disabled server-wide — refuses passkey login entirely", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = false
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.Error(t, err, "must refuse rather than silently issue a token with no compatible second factor available")
		assert.Nil(t, authRes)
	})
}
