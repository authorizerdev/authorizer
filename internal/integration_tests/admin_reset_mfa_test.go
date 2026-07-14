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

// TestAdminResetMFA covers the _update_user{reset_mfa: true} admin recovery
// path: it must clear mfa_locked_at, is_multi_factor_auth_enabled, and
// has_skipped_mfa_setup_at, and delete every enrolled authenticator and
// webauthn credential row for the user.
func TestAdminResetMFA(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	email := "admin_reset_mfa_" + uuid.NewString() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	// cfg.EnableMFA/EnableTOTPLogin are both on here, so SignUp itself now
	// runs the same MFA gate as Login (Task 7): its response withholds the
	// token and the User field (matching login.go's own mfaGateOfferAll/
	// BlockEnroll responses, which never set User either). Look the user up
	// by email instead of relying on a User field that isn't there for this
	// path.
	signedUpUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	userID := signedUpUser.ID

	// Put the user into the exact state reset_mfa is meant to unwind: locked,
	// MFA enabled, skip recorded, plus a real TOTP authenticator row and a
	// webauthn credential row.
	now := time.Now().Unix()
	user, err := ts.StorageProvider.GetUserByID(ctx, userID)
	require.NoError(t, err)
	user.MFALockedAt = &now
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	user.HasSkippedMFASetupAt = &now
	_, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)

	_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
		UserID:     userID,
		Method:     constants.EnvKeyTOTPAuthenticator,
		Secret:     "test-secret",
		VerifiedAt: &now,
	})
	require.NoError(t, err)

	_, err = ts.StorageProvider.AddWebauthnCredential(ctx, &schemas.WebauthnCredential{
		UserID:       userID,
		CredentialID: uuid.NewString(),
		PublicKey:    "test-public-key",
	})
	require.NoError(t, err)

	// sanity-check the fixture actually landed before asserting the reset.
	authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	creds, err := ts.StorageProvider.ListWebauthnCredentialsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, creds, 1)

	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	updateRes, err := ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
		ID:       userID,
		ResetMfa: refs.NewBoolRef(true),
	})
	require.NoError(t, err)
	require.NotNil(t, updateRes)

	reset, err := ts.StorageProvider.GetUserByID(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, reset.MFALockedAt, "reset_mfa must clear mfa_locked_at")
	assert.Nil(t, reset.IsMultiFactorAuthEnabled, "reset_mfa must clear is_multi_factor_auth_enabled")
	assert.Nil(t, reset.HasSkippedMFASetupAt, "reset_mfa must clear has_skipped_mfa_setup_at")

	_, err = ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	assert.Error(t, err, "reset_mfa must delete the user's authenticator rows")

	remainingCreds, err := ts.StorageProvider.ListWebauthnCredentialsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, remainingCreds, "reset_mfa must delete the user's webauthn credentials")
}
