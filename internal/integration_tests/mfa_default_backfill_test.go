package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestMFADefaultBackfillOnLogin guards the lazy migration for accounts that
// existed before the MFA-default-on-signup behavior shipped (or before MFA
// was ever configured on this server): the first time such a user
// authenticates, IsMultiFactorAuthEnabled - if never explicitly set - is
// backfilled to match EnableMFA and persisted, so existing accounts converge
// to the same offer-with-skip behavior a new signup already gets.
func TestMFADefaultBackfillOnLogin(t *testing.T) {
	const password = "Password@123"

	t.Run("password login backfills an existing user with MFA available", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "mfa_backfill_login_" + uuid.New().String() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		// Simulate an account that predates the signup default: force the
		// flag back to nil, as if it were created before this feature shipped.
		user.IsMultiFactorAuthEnabled = nil
		user, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		require.Nil(t, user.IsMultiFactorAuthEnabled)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: user.Email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.AccessToken, "optional MFA must not block login on the backfilling login")
		assert.True(t, refs.BoolValue(res.ShouldOfferMfaSetup), "the newly-backfilled user should be offered setup, same as a fresh signup")

		reloaded, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, refs.BoolValue(reloaded.IsMultiFactorAuthEnabled), "the backfill must be persisted, not just applied in-memory for this one response")
	})

	t.Run("password login does not backfill when MFA is unavailable", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = false
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "mfa_backfill_none_" + uuid.New().String() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)

		reloaded, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Nil(t, reloaded.IsMultiFactorAuthEnabled, "must not backfill when MFA isn't available at all")
	})

	t.Run("does not override an explicit false", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "mfa_backfill_explicit_false_" + uuid.New().String() + "@authorizer.dev"
		explicit := false
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
			IsMultiFactorAuthEnabled: &explicit,
		})
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, refs.BoolValue(res.ShouldOfferMfaSetup))

		reloaded, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.False(t, refs.BoolValue(reloaded.IsMultiFactorAuthEnabled), "an explicit false must never be silently overridden")
	})
}

// TestMFADefaultBackfillOnPasskeyLogin covers the same lazy-backfill guard
// through the passkey primary-login entry point, so a user who authenticates
// exclusively via passkey converges to the same default as a password-login
// user rather than being permanently excluded from the offer.
func TestMFADefaultBackfillOnPasskeyLogin(t *testing.T) {
	t.Run("passkey primary login backfills an existing user with MFA available", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = nil
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes.AccessToken, "EnforceMFA is off, so passkey login must still succeed directly")

		reloaded, err := ts.StorageProvider.GetUserByID(t.Context(), user.ID)
		require.NoError(t, err)
		assert.True(t, refs.BoolValue(reloaded.IsMultiFactorAuthEnabled))
	})
}
