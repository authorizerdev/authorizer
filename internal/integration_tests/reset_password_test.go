package integration_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure imports are used
var _ = time.Now
var _ = fmt.Sprintf
var _ = strings.Contains

// TestResetPassword tests the reset password functionality
func TestResetPassword(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "reset_password_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, signupRes)
	assert.NotNil(t, signupRes.User)

	// Create forgot password request
	t.Run("should fail for invalid request", func(t *testing.T) {
		resetPasswordReq := &model.ResetPasswordRequest{
			Token:           refs.NewStringRef("test"),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.Error(t, err)
		assert.Nil(t, forgotPasswordRes)
	})

	t.Run("should fail for password mismatch", func(t *testing.T) {
		resetPasswordReq := &model.ResetPasswordRequest{
			Token:           refs.NewStringRef("test"),
			Password:        "NewPassword@123",
			ConfirmPassword: "DifferentPassword@123",
		}
		res, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail reset password with expired OTP", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.IsSMSServiceEnabled = true
		cfg2.EnableMobileBasicAuthentication = true
		cfg2.EnablePhoneVerification = true
		ts2 := initTestSetup(t, cfg2)
		req2, ctx2 := createContext(ts2)

		mobile := "+14155550199"
		signupReq2 := &model.SignUpRequest{
			PhoneNumber:     &mobile,
			Password:        password,
			ConfirmPassword: password,
		}
		_, err := ts2.GraphQLProvider.SignUp(ctx2, signupReq2)
		require.NoError(t, err)

		// Get user and OTP
		user, err := ts2.StorageProvider.GetUserByPhoneNumber(ctx2, mobile)
		require.NoError(t, err)

		otpData, err := ts2.StorageProvider.GetOTPByPhoneNumber(ctx2, mobile)
		require.NoError(t, err)

		// Set OTP to expired
		expiredOTP := &schemas.OTP{
			ID:          otpData.ID,
			Email:       otpData.Email,
			PhoneNumber: otpData.PhoneNumber,
			Otp:         otpData.Otp,
			ExpiresAt:   time.Now().Add(-10 * time.Minute).Unix(),
		}
		_, err = ts2.StorageProvider.UpsertOTP(ctx2, expiredOTP)
		require.NoError(t, err)

		// Get MFA session
		allData, err := ts2.MemoryStoreProvider.GetAllData()
		require.NoError(t, err)
		sessionKey := ""
		for k := range allData {
			if strings.Contains(k, user.ID) {
				splitData := strings.Split(k, ":")
				if len(splitData) > 1 {
					sessionKey = splitData[1]
					break
				}
			}
		}
		req2.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", sessionKey))

		resetReq := &model.ResetPasswordRequest{
			Otp:             refs.NewStringRef(otpData.Otp),
			PhoneNumber:     refs.NewStringRef(mobile),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}
		res, err := ts2.GraphQLProvider.ResetPassword(ctx2, resetReq)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("should reset password with verification token", func(t *testing.T) {
		forgotPasswordReq := &model.ForgotPasswordRequest{
			Email: refs.NewStringRef(email),
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ForgotPassword(ctx, forgotPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, forgotPasswordRes)
		assert.NotEmpty(t, forgotPasswordRes.Message)

		// Validate if the entry is created in db
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.NotEmpty(t, request.Token)
		assert.Equal(t, email, request.Email)

		// Reset password using the token
		resetPasswordReq := &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(request.Token),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}

		resetPasswordRes, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, resetPasswordRes)
		assert.NotEmpty(t, resetPasswordRes.Message)

		// Validate if the password is updated in db by logging in
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: "NewPassword@123",
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.NotNil(t, loginRes.AccessToken)
	})
}
