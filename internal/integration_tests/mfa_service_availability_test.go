package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestMFAServiceAvailability verifies that enabling a user's MFA via the admin
// UpdateUser path is gated on the instance actually being able to do MFA (the
// same criteria login uses), that _admin_meta reports that availability, and
// that disabling MFA is always allowed regardless.
func TestMFAServiceAvailability(t *testing.T) {
	t.Run("service disabled: admin_meta false, enable rejected, disable allowed", func(t *testing.T) {
		cfg := getTestConfig() // no MFA flags -> service unavailable
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		setAdminCookie(t, ts)
		meta, err := ts.GraphQLProvider.AdminMeta(ctx)
		require.NoError(t, err)
		require.False(t, meta.IsMultiFactorAuthServiceEnabled, "no MFA method configured")

		email := "mfa_off_" + uuid.New().String() + "@authorizer.dev"
		su, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: "Password@123", ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)

		// Enabling MFA is rejected when no MFA service is available.
		setAdminCookie(t, ts)
		_, err = ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
			ID: su.User.ID, IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.Error(t, err)

		// Force the user MFA-enabled directly, then confirm disabling is allowed
		// even though the service is off (an admin must be able to turn it off).
		u, err := ts.StorageProvider.GetUserByID(ctx, su.User.ID)
		require.NoError(t, err)
		u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, u)
		require.NoError(t, err)

		setAdminCookie(t, ts)
		res, err := ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
			ID: su.User.ID, IsMultiFactorAuthEnabled: refs.NewBoolRef(false),
		})
		require.NoError(t, err)
		require.False(t, refs.BoolValue(res.IsMultiFactorAuthEnabled))
	})

	t.Run("service enabled (mfa+totp): admin_meta true, enable succeeds", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		setAdminCookie(t, ts)
		meta, err := ts.GraphQLProvider.AdminMeta(ctx)
		require.NoError(t, err)
		require.True(t, meta.IsMultiFactorAuthServiceEnabled)

		email := "mfa_on_" + uuid.New().String() + "@authorizer.dev"
		su, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: "Password@123", ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)

		setAdminCookie(t, ts)
		res, err := ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
			ID: su.User.ID, IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.True(t, refs.BoolValue(res.IsMultiFactorAuthEnabled))
	})
}
