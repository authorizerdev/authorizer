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
