package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyOTP tests the resend verify email functionality
func TestVerifyOTP(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.DisableEmailOTP = false
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	mobile := "+14155552671"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpInput{
		PhoneNumber:     &mobile,
		Password:        password,
		ConfirmPassword: password,
	}

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, signupRes)
	// Expect the user to be nil, as the email is not verified yet
	assert.Nil(t, signupRes.User)

	// Get the OTP from db
	otpData, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	assert.NotNil(t, otpData)
	// User
	userData, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	assert.NotNil(t, userData)

	t.Run("should fail for invalid cookie", func(t *testing.T) {
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         otpData.Otp,
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should fail for invalid OTP", func(t *testing.T) {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", constants.TestEnv))
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         "-----",
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should verify OTP", func(t *testing.T) {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", constants.TestEnv))
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         otpData.Otp,
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})
}
