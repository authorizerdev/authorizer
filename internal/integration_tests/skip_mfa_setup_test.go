package integration_tests

import (
	"errors"
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
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestSkipMFASetup covers the two security-relevant behaviors of the
// skip_mfa_setup mutation:
//   - with a valid token and MFA optional, it records HasSkippedMFASetupAt
//     and a subsequent login no longer offers setup (should_offer_mfa_setup
//     is false).
//   - with EnforceMFA=true it is rejected with KindFailedPrecondition even
//     though the caller is authenticated — enforcement is never skippable,
//     and this must be re-checked server-side regardless of what the
//     client believes the gate state to be.
func TestSkipMFASetup(t *testing.T) {
	const password = "Password@123"

	t.Run("skips setup when MFA is optional and quiets a later login", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "skip_mfa_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, loginRes.AccessToken)
		assert.True(t, refs.BoolValue(loginRes.ShouldOfferMfaSetup), "first login with optional MFA and no prior enrollment/skip must offer setup")

		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)
		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx)
		require.NoError(t, err)
		require.NotNil(t, skipRes)
		assert.Equal(t, "MFA setup skipped", skipRes.Message)

		updated, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, updated.HasSkippedMFASetupAt, "skip_mfa_setup must persist HasSkippedMFASetupAt")

		ts.GinContext.Request.Header.Set("Authorization", "")
		secondLogin, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, secondLogin.AccessToken)
		assert.False(t, refs.BoolValue(secondLogin.ShouldOfferMfaSetup), "must not nag a user who already skipped setup")
	})

	t.Run("rejects with FailedPrecondition when MFA is enforced, even with a valid token", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
		req, ctx := createContext(ts)

		// Mint a genuinely valid access token the same way a real enforced-MFA
		// user would end up with one: complete TOTP enrollment and verify it
		// via the real VerifyOTP path (mirrors
		// TestVerifyOTPTOTPThroughService), rather than fabricating a token.
		// This proves the EnforceMFA rejection below is a true server-side
		// re-check on an authenticated caller, not an artifact of a missing
		// or invalid token.
		email := "skip_mfa_enforced_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)
		require.NotEmpty(t, authConfig.Secret)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		code, err := totp.GenerateCode(authConfig.Secret, time.Now())
		require.NoError(t, err)
		verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email:  &email,
			Otp:    code,
			IsTotp: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.NotNil(t, verifyRes)
		require.NotEmpty(t, verifyRes.AccessToken, "a valid TOTP passcode must mint a real access token")

		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)
		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx)
		require.Error(t, err)
		assert.Nil(t, skipRes)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindFailedPrecondition, svcErr.Kind, "EnforceMFA must reject with FailedPrecondition, not Unauthenticated or any other kind")
	})
}
