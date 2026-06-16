package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestVerifyOTPTOTPThroughService exercises the TOTP / recovery-code branch of
// VerifyOTP end-to-end through the migrated service layer (the GraphQL resolver
// now delegates to service.VerifyOTP, which calls p.AuthenticatorProvider).
// This is the regression guard for the AuthenticatorProvider being wired into
// the service Dependencies — without it this path nil-panics. It also confirms
// a valid TOTP passcode and a recovery code both complete authentication.
func TestVerifyOTPTOTPThroughService(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
	req, ctx := createContext(ts)

	// A verified user we can attach a TOTP authenticator to.
	email := "verify_totp_" + uuid.NewString() + "@authorizer.dev"
	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef(email),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, authConfig.Secret)
	require.NotEmpty(t, authConfig.RecoveryCodes)

	// Establish the MFA session the verify step requires: a cookie value that
	// also exists in the memory store under the user id (verify_otp checks
	// GetMfaSession(user.ID, <cookie value>)).
	armMfaSession := func() {
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession,
			time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
	}

	t.Run("valid TOTP passcode completes auth", func(t *testing.T) {
		armMfaSession()
		code, err := totp.GenerateCode(authConfig.Secret, time.Now())
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email:  &email,
			Otp:    code,
			IsTotp: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken, "a valid TOTP passcode must mint an access token")
	})

	t.Run("valid recovery code completes auth", func(t *testing.T) {
		armMfaSession()
		res, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email:  &email,
			Otp:    authConfig.RecoveryCodes[0],
			IsTotp: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken, "a valid recovery code must mint an access token")
	})

	t.Run("invalid TOTP passcode is rejected", func(t *testing.T) {
		armMfaSession()
		res, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email:  &email,
			Otp:    "000000",
			IsTotp: refs.NewBoolRef(true),
		})
		require.Error(t, err)
		assert.Nil(t, res)
	})
}
