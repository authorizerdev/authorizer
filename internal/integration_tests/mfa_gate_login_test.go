package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestLoginMFAGateTokenWithholding is the regression guard for the security
// property described in mfa_gate.go and wired into login.go's TOTP branch:
// mfaGateBlockVerify and mfaGateBlockEnroll must NEVER reach the code path
// that sets AccessToken on the login response, while mfaGateNone,
// mfaGateOfferSetup and mfaGateSkippedSetup must all fall through to normal
// token issuance.
//
// mfa_gate_test.go already covers resolveMFAGate's pure decision table in
// isolation. This test drives the same 5 outcomes through the real
// login.go switch (via GraphQLProvider.Login, a thin wrapper around
// service.Provider.Login) with a user/config combination engineered to land
// on exactly one outcome, and asserts on AccessToken directly. A future edit
// that removes a `return` from the mfaGateBlockVerify/mfaGateBlockEnroll
// cases — or that lets one of them fall through — would compile and pass
// TestResolveMFAGate unchanged, but would fail here because AccessToken
// stops being empty.
func TestLoginMFAGateTokenWithholding(t *testing.T) {
	const password = "Password@123"

	// signUpUser creates a fresh, auto-verified basic-auth user (email
	// verification is off in getTestConfig) via the real SignUp path, so the
	// stored password hash is one login.go's bcrypt check actually accepts.
	signUpUser := func(t *testing.T, ts *testSetup, ctx context.Context) *schemas.User {
		t.Helper()
		email := "mfa_gate_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		return user
	}

	// addVerifiedAuthenticator gives the user a completed TOTP authenticator,
	// the condition login.go reads as authenticatorVerified=true.
	addVerifiedAuthenticator := func(t *testing.T, ts *testSetup, ctx context.Context, userID string) {
		t.Helper()
		now := time.Now().Unix()
		_, err := ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     userID,
			Method:     constants.EnvKeyTOTPAuthenticator,
			Secret:     "dummy-secret-for-gate-test",
			VerifiedAt: &now,
		})
		require.NoError(t, err)
	}

	t.Run("mfaGateBlockVerify withholds the token", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := signUpUser(t, ts, ctx)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		user, err := ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		addVerifiedAuthenticator(t, ts, ctx, user.ID)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "a user with a verified authenticator must not receive a token before verifying it")
		assert.True(t, refs.BoolValue(res.ShouldShowTotpScreen))
	})

	t.Run("mfaGateBlockEnroll withholds the token", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := signUpUser(t, ts, ctx)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		user, err := ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		// No authenticator enrolled yet: enforceMFA must force enrollment.

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "an org-enforced user who hasn't finished enrollment must not receive a token")
		assert.True(t, refs.BoolValue(res.ShouldShowTotpScreen))
		assert.NotNil(t, res.AuthenticatorSecret, "block-enroll must hand back a fresh enrollment payload")
	})

	t.Run("mfaGateOfferSetup issues a token and offers setup", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := signUpUser(t, ts, ctx)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		user, err := ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		// Not enrolled, MFA not enforced, never skipped before.

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.AccessToken, "optional MFA must not block login")
		assert.NotEmpty(t, *res.AccessToken)
		assert.True(t, refs.BoolValue(res.ShouldOfferMfaSetup))
		assert.NotNil(t, res.AuthenticatorSecret)
	})

	t.Run("mfaGateSkippedSetup issues a token quietly", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := signUpUser(t, ts, ctx)
		skippedAt := time.Now().Unix()
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		user.HasSkippedMFASetupAt = &skippedAt
		user, err := ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.AccessToken, "a user who already skipped setup must still be able to log in")
		assert.NotEmpty(t, *res.AccessToken)
		assert.False(t, refs.BoolValue(res.ShouldOfferMfaSetup), "must not nag a user who already skipped setup")
	})

	t.Run("mfaGateNone issues a token normally", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := signUpUser(t, ts, ctx)
		// IsMultiFactorAuthEnabled left false/unset: the gate is a no-op.

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.AccessToken)
		assert.NotEmpty(t, *res.AccessToken)
		assert.False(t, refs.BoolValue(res.ShouldShowTotpScreen))
		assert.False(t, refs.BoolValue(res.ShouldOfferMfaSetup))
	})
}
