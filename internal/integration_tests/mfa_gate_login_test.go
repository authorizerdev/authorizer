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
// mfaGateBlockVerify, mfaGateBlockEnroll, and mfaGateOfferAll must NEVER
// reach the code path that sets AccessToken on the login response, while
// mfaGateNone and mfaGateSkippedSetup must fall through to normal token
// issuance.
//
// mfa_gate_test.go already covers resolveMFAGate's pure decision table in
// isolation. This test drives the same 5 outcomes through the real
// login.go switch (via GraphQLProvider.Login, a thin wrapper around
// service.Provider.Login) with a user/config combination engineered to land
// on exactly one outcome, and asserts on AccessToken directly. A future edit
// that removes a `return` from the mfaGateBlockVerify/mfaGateBlockEnroll/
// mfaGateOfferAll cases — or that lets one of them fall through — would
// compile and pass TestResolveMFAGate unchanged, but would fail here
// because AccessToken stops being empty.
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
	// the condition login.go reads as authenticatorVerified=true. Upserts
	// rather than blindly inserting: SignUp itself now runs the same MFA
	// gate as Login (Task 7), so signUpUser (below, with cfg.EnableMFA=true)
	// already leaves an unverified TOTP row behind via its own
	// generateTOTPEnrollment call. StorageProvider.AddAuthenticator no-ops
	// when a row already exists for (userID, method), so calling it here
	// unconditionally would silently fail to mark that pre-existing row
	// verified.
	addVerifiedAuthenticator := func(t *testing.T, ts *testSetup, ctx context.Context, userID string) {
		t.Helper()
		now := time.Now().Unix()
		existing, _ := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
		if existing != nil {
			existing.Secret = "dummy-secret-for-gate-test"
			existing.VerifiedAt = &now
			_, err := ts.StorageProvider.UpdateAuthenticator(ctx, existing)
			require.NoError(t, err)
			return
		}
		_, err := ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     userID,
			Method:     constants.EnvKeyTOTPAuthenticator,
			Secret:     "dummy-secret-for-gate-test",
			VerifiedAt: &now,
		})
		require.NoError(t, err)
	}

	// addWebauthnCredential gives the user a registered passkey, the
	// condition login.go reads (via ListWebauthnCredentialsByUserID) as an
	// alternative authenticatorVerified=true source alongside TOTP.
	addWebauthnCredential := func(t *testing.T, ts *testSetup, ctx context.Context, userID string) {
		t.Helper()
		_, err := ts.StorageProvider.AddWebauthnCredential(ctx, &schemas.WebauthnCredential{
			UserID:       userID,
			CredentialID: uuid.NewString(),
			PublicKey:    "dummy-public-key-for-gate-test",
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
		assert.False(t, refs.BoolValue(res.ShouldOfferWebauthnMfaVerify), "a TOTP-only user must not be offered a passkey verify option they never registered")
	})

	t.Run("mfaGateBlockVerify offers passkey verify for a passkey-only user", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := signUpUser(t, ts, ctx)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		user, err := ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		addWebauthnCredential(t, ts, ctx, user.ID)
		// No TOTP authenticator enrolled — passkey is this user's only factor.

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "a user with a registered passkey must not receive a token before verifying it")
		assert.False(t, refs.BoolValue(res.ShouldShowTotpScreen), "must not force a TOTP screen on a user who never enrolled TOTP")
		assert.True(t, refs.BoolValue(res.ShouldOfferWebauthnMfaVerify))
	})

	t.Run("mfaGateBlockVerify offers both methods for a dual-enrolled user", func(t *testing.T) {
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
		addWebauthnCredential(t, ts, ctx, user.ID)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken)
		assert.True(t, refs.BoolValue(res.ShouldShowTotpScreen))
		assert.True(t, refs.BoolValue(res.ShouldOfferWebauthnMfaVerify))
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

	t.Run("mfaGateOfferAll withholds the token and offers every available method", func(t *testing.T) {
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
		assert.Nil(t, res.AccessToken, "a first-time optional-MFA offer must withhold the token until setup or skip")
		assert.True(t, refs.BoolValue(res.ShouldShowTotpScreen))
		assert.NotNil(t, res.AuthenticatorSecret, "offer-all must hand back a fresh TOTP enrollment payload")
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
		// Signup now defaults IsMultiFactorAuthEnabled to true whenever MFA
		// is available server-wide (see signup.go), so this test's actual
		// target state - a user for whom MFA is individually off - must be
		// set explicitly rather than relying on signup to leave it unset.
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(false)
		user, err := ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.AccessToken)
		assert.NotEmpty(t, *res.AccessToken)
		assert.False(t, refs.BoolValue(res.ShouldShowTotpScreen))
		assert.False(t, refs.BoolValue(res.ShouldOfferMfaSetup))
	})
}
