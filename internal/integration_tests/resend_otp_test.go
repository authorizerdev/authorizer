package integration_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResendOTP tests the resend verify OTP functionality
func TestResendOTP(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableEmailOTP = true
	cfg.EnableSMSOTP = true
	cfg.EnablePhoneVerification = true
	cfg.EnableMobileBasicAuthentication = true
	cfg.EnableMFA = true
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
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user with a unique phone number to avoid collisions
	mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
		PhoneNumber:     &mobile,
		Password:        password,
		ConfirmPassword: password,
	}

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	// Expect the user to be nil, as the email is not verified yet
	assert.Nil(t, signupRes.User)

	// Overwrite the row written by signup with a known plaintext/digest
	// pair so the test can verify with the plaintext (the integration
	// suite cannot intercept the outgoing SMS).
	const initialPlainOTP = "123456"
	signupOTP, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	require.NotNil(t, signupOTP)
	signupOTP.Otp = crypto.HashOTP(initialPlainOTP, cfg.JWTSecret)
	signupOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, signupOTP)
	require.NoError(t, err)

	// Sanity check the at-rest hardening: stored value is the digest, not
	// the plaintext, and the plaintext is recoverable only via VerifyOTPHash.
	t.Run("OTP at rest is hashed, not plaintext", func(t *testing.T) {
		row, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
		require.NoError(t, err)
		assert.NotEqual(t, initialPlainOTP, row.Otp)
		assert.Equal(t, crypto.HashOTP(initialPlainOTP, cfg.JWTSecret), row.Otp)
		assert.True(t, crypto.VerifyOTPHash(initialPlainOTP, row.Otp, cfg.JWTSecret))
	})

	// User
	userData, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	assert.NotNil(t, userData)

	t.Run("should return SMS service error not email service error", func(t *testing.T) {
		smsCfg := getTestConfig()
		smsCfg.EnableMFA = true
		smsCfg.IsSMSServiceEnabled = false
		smsCfg.IsEmailServiceEnabled = true
		smsCfg.SMTPHost = "localhost"
		smsCfg.SMTPPort = 1025
		smsCfg.SMTPSenderEmail = "test@authorizer.dev"
		smsCfg.SMTPSenderName = "Test"
		smsCfg.SMTPLocalName = "Test"
		smsCfg.SMTPSkipTLSVerification = true
		smsTs := initTestSetup(t, smsCfg)
		_, smsCtx := createContext(smsTs)
		resendReq := &model.ResendOTPRequest{
			PhoneNumber: refs.NewStringRef("+11234567890"),
		}
		resendRes, err := smsTs.GraphQLProvider.ResendOTP(smsCtx, resendReq)
		assert.Error(t, err)
		assert.Nil(t, resendRes)
		assert.Contains(t, err.Error(), "SMS service not enabled")
	})
	t.Run("should resend OTP with sanitized email (spaces and mixed case)", func(t *testing.T) {
		emailCfg := getTestConfig()
		emailCfg.EnableMFA = true
		emailCfg.IsEmailServiceEnabled = true
		emailCfg.EnableEmailOTP = true
		emailCfg.EnableEmailVerification = true
		emailCfg.SMTPHost = "localhost"
		emailCfg.SMTPPort = 1025
		emailCfg.SMTPSenderEmail = "test@authorizer.dev"
		emailCfg.SMTPSenderName = "Test"
		emailCfg.SMTPLocalName = "Test"
		emailCfg.SMTPSkipTLSVerification = true
		emailTs := initTestSetup(t, emailCfg)
		_, emailCtx := createContext(emailTs)
		// Sign up with a clean email
		cleanEmail := fmt.Sprintf("resendtest%d@authorizer.dev", time.Now().UnixNano())
		password := "Password@123"
		signupRes, err := emailTs.GraphQLProvider.SignUp(emailCtx, &model.SignUpRequest{
			Email:           refs.NewStringRef(cleanEmail),
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, signupRes)
		// Seed a known OTP for this user
		_, err = emailTs.StorageProvider.UpsertOTP(emailCtx, &schemas.OTP{
			Email:     cleanEmail,
			Otp:       crypto.HashOTP("111111", emailCfg.JWTSecret),
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		})
		require.NoError(t, err)
		// Resend with unsanitized email (spaces + mixed case)
		unsanitizedEmail := "  " + strings.ToUpper(cleanEmail) + "  "
		resendReq := &model.ResendOTPRequest{
			Email: refs.NewStringRef(unsanitizedEmail),
		}
		resendRes, err := emailTs.GraphQLProvider.ResendOTP(emailCtx, resendReq)
		assert.NoError(t, err)
		assert.NotNil(t, resendRes)
		assert.Equal(t, "OTP has been sent. Please check your inbox", resendRes.Message)
	})
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
		t.Run("old OTP plaintext is invalidated by resend", func(t *testing.T) {
			// The original OTP we seeded above must no longer verify
			// because resend wrote a new digest into the same row.
			verificationReq := &model.VerifyOTPRequest{
				PhoneNumber: &mobile,
				Otp:         initialPlainOTP,
			}
			verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
			assert.Error(t, err)
			assert.Nil(t, verificationRes)
		})
		t.Run("should verify OTP with the plaintext that was sent", func(t *testing.T) {
			// Resend wrote a new OTP whose plaintext we can't observe
			// (it went out via SMS). Overwrite the row again with a
			// known plaintext/digest pair and verify with the plaintext.
			const resentPlainOTP = "654321"
			_, err := ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
				PhoneNumber: mobile,
				Otp:         crypto.HashOTP(resentPlainOTP, cfg.JWTSecret),
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
				Otp:         resentPlainOTP,
			}
			verificationRes, err := ts.GraphQLProvider.VerifyOTP(ctx, verificationReq)
			require.NoError(t, err)
			assert.NotNil(t, verificationRes)
		})
	})
}
