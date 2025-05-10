package integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResendOTP tests the resend verify OTP functionality
func TestResendOTP(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.DisableEmailOTP = false
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	mobile := "+14155552672"
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

	t.Run("should fail if request for given email or phone number does not exists", func(t *testing.T) {
		resendReq := &model.ResendOTPRequest{
			PhoneNumber: refs.NewStringRef("2131231212"),
		}
		resendRes, err := ts.GraphQLProvider.ResendOTP(ctx, resendReq)
		assert.Error(t, err)
		assert.Nil(t, resendRes)
	})
	t.Run("should send resend request{mobile}", func(t *testing.T) {
		resendReq := &model.ResendOTPRequest{
			PhoneNumber: refs.NewStringRef(mobile),
		}
		resendRes, err := ts.GraphQLProvider.ResendOTP(ctx, resendReq)
		assert.NoError(t, err)
		assert.NotNil(t, resendRes)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", constants.TestEnv))
		t.Run("old OTP should be invalidated", func(t *testing.T) {
			verificationReq := &model.VerifyOTPRequest{
				PhoneNumber: &mobile,
				Otp:         otpData.Otp,
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
			newOtpData, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
			require.NoError(t, err)
			assert.NotNil(t, newOtpData)
			verificationReq := &model.VerifyOTPRequest{
				PhoneNumber: &mobile,
				Otp:         newOtpData.Otp,
			}
			verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
			require.NoError(t, err)
			assert.NotNil(t, verificationRes)
		})
	})
}
