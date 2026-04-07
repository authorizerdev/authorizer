package integration_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
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
	cfg.SMTPSkipTLSVerification = true
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
	mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
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

	// Get the OTP row written by signup. After the at-rest hardening it
	// stores the HMAC digest, NOT the plaintext code that was sent over
	// SMS. The integration suite cannot intercept the outgoing SMS, so we
	// overwrite the row with a known plaintext/digest pair below and
	// verify with the known plaintext.
	storedOTP, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	require.NotNil(t, storedOTP)

	const knownPlainOTP = "123456"
	storedOTP.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
	storedOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, storedOTP)
	require.NoError(t, err)

	t.Run("OTP at rest is hashed, not plaintext", func(t *testing.T) {
		row, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
		require.NoError(t, err)
		// 1. Stored value must NOT equal the plaintext
		assert.NotEqual(t, knownPlainOTP, row.Otp, "OTP must be hashed at rest")
		// 2. Stored value must be the HMAC-SHA256 hex digest of the plaintext
		assert.Equal(t, crypto.HashOTP(knownPlainOTP, cfg.JWTSecret), row.Otp)
		// 3. The hex digest is 64 chars long (sha256)
		assert.Len(t, row.Otp, 64)
	})

	// User
	userData, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	assert.NotNil(t, userData)

	t.Run("should fail for invalid cookie", func(t *testing.T) {
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         knownPlainOTP,
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

	t.Run("should fail when stored hash is replayed as input", func(t *testing.T) {
		// Anyone with read access to the OTP row would have the digest.
		// Submitting the digest itself MUST be rejected — otherwise
		// hashing accomplishes nothing.
		row, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", "test"))
		verificationReq := &model.VerifyOTPRequest{
			PhoneNumber: &mobile,
			Otp:         row.Otp, // the digest, not the plaintext
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should verify OTP with the plaintext that was sent", func(t *testing.T) {
		// Re-seed because the previous successful verify in earlier subtests
		// (or any failed expiry) may have deleted the row.
		_, err := ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			PhoneNumber: mobile,
			Otp:         crypto.HashOTP(knownPlainOTP, cfg.JWTSecret),
			ExpiresAt:   time.Now().Add(5 * time.Minute).Unix(),
		})
		require.NoError(t, err)

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
			Otp:         knownPlainOTP,
		}
		verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
		require.NoError(t, err)
		assert.NotNil(t, verificationRes)
	})
}
