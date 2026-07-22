package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyOTP_SessionPurpose is the regression test for the MFA-downgrade
// finding: a password-reset OTP session must never be redeemable for a token
// via VerifyOTP, and VerifyOTP must keep accepting the ordinary Challenge
// session that ResendOTP legitimately hands off to it.
func TestVerifyOTP_SessionPurpose(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	cfg.EnablePhoneVerification = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		PhoneNumber:     &mobile,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
	require.NoError(t, err)

	const knownPlainOTP = "654321"
	seedOTP := func() {
		otpData, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
		require.NoError(t, err)
		otpData.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
		otpData.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
		_, err = ts.StorageProvider.UpsertOTP(ctx, otpData)
		require.NoError(t, err)
	}

	t.Run("a password_reset session cannot be redeemed via VerifyOTP", func(t *testing.T) {
		seedOTP()
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposePasswordReset, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         knownPlainOTP,
		})
		assert.Error(t, err, "a password_reset-purpose session must not be accepted by VerifyOTP")
		assert.Nil(t, verificationRes)
	})

	t.Run("a challenge session (ResendOTP-style) is still accepted by VerifyOTP", func(t *testing.T) {
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         "000000",
		})
		// Still reaches OTP validation (wrong code here) rather than being
		// rejected at the session-purpose gate — proves Challenge still works.
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
		assert.NotContains(t, err.Error(), "invalid session")
	})

	t.Run("an unresolvable/absent session is rejected", func(t *testing.T) {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", uuid.NewString()))
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         "000000",
		})
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})
}
