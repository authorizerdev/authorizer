package integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyOTP tests the resend verify OTP functionality
func TestVerifyOTP(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableEmailOTP = true
	cfg.EnableSMSOTP = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	cfg.IsSMSServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.TwilioAPISecret = "test-twilio-api-secret"
	cfg.TwilioAPIKey = "test-twilio-api-key"
	cfg.TwilioAccountSID = "test-twilio-account-sid"
	cfg.TwilioSender = "test-twilio-sender"
	cfg.EnableMobileBasicAuthentication = true
	cfg.EnablePhoneVerification = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	mobile := "+14155552671"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
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
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", "test"))
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         "-----",
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should verify OTP", func(t *testing.T) {
		// Get MFA session cookie
		allData, err := ts.MemoryStoreProvider.GetAllData()
		require.NoError(t, err)
		sessionKey := ""
		for k := range allData {
			if strings.Contains(k, userData.ID) {
				splitData := strings.Split(k, ":")
				if len(splitData) > 1 {
					sessionKey = splitData[1]
					break
				}
			}
		}
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", sessionKey))
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         otpData.Otp,
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		require.NoError(t, err)
		assert.NotNil(t, verificationRes)
	})
}
