package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestAdminUpdateUserEnforceMFA verifies EnforceMFA is absolute on the admin
// path: an admin cannot persist IsMultiFactorAuthEnabled=false while the org
// enforces MFA, matching the self-service update_profile.go guard.
func TestAdminUpdateUserEnforceMFA(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	cfg.EnforceMFA = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	email := "admin_enforce_mfa_" + uuid.NewString() + "@authorizer.dev"
	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:                    refs.NewStringRef(email),
		EmailVerifiedAt:          &now,
		SignupMethods:            constants.AuthRecipeMethodBasicAuth,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
	})
	require.NoError(t, err)

	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	_, err = ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
		ID:                       user.ID,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(false),
	})
	require.Error(t, err, "admin must not be able to disable MFA while EnforceMFA is on")

	persisted, err := ts.StorageProvider.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, refs.BoolValue(persisted.IsMultiFactorAuthEnabled), "MFA must remain enabled after a rejected disable")
}

// TestAdminUpdateUserMFAFlagNilToFalse is a regression test: the "did the flag
// actually change" check used to compare refs.BoolValue(user...) !=
// refs.BoolValue(params...), and BoolValue(nil) == false, so a user whose
// IsMultiFactorAuthEnabled was never set (nil) compared as "false != false"
// against an explicit false — no change detected — and the assignment was
// silently skipped, leaving the flag stuck at nil instead of the requested
// false.
func TestAdminUpdateUserMFAFlagNilToFalse(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	email := "admin_mfa_nil_to_false_" + uuid.NewString() + "@authorizer.dev"
	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef(email),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		// IsMultiFactorAuthEnabled deliberately left nil (unset).
	})
	require.NoError(t, err)
	require.Nil(t, user.IsMultiFactorAuthEnabled, "fixture must start unset to exercise the nil->false transition")

	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	_, err = ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
		ID:                       user.ID,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(false),
	})
	require.NoError(t, err)

	persisted, err := ts.StorageProvider.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted.IsMultiFactorAuthEnabled, "explicit false must be persisted, not left as nil")
	assert.False(t, refs.BoolValue(persisted.IsMultiFactorAuthEnabled))
}
