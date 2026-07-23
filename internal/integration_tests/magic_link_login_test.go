package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestMagicLinkLogin tests the magic link login functionality of the Authorizer application.
func TestMagicLinkLogin(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMagicLinkLogin = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	cfg.IsSMSServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "magic_link_user" + uuid.New().String() + "@authorizer.dev"

	t.Run("should fail for missing email", func(t *testing.T) {
		loginReq := &model.MagicLinkLoginRequest{}
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	t.Run("should fail for invalid email", func(t *testing.T) {
		loginReq := &model.MagicLinkLoginRequest{
			Email: "invalid-email",
		}
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	t.Run("should pass for valid email", func(t *testing.T) {
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
			Email: email,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.Message)

		verificationRequest, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
		verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, *verifyRes.AccessToken)

		// Set the Authorization header for the Profile request
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)

		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, profile)

		// Clean up the header after the test
		ts.GinContext.Request.Header.Set("Authorization", "")
	})
}

// TestMagicLinkLoginEmailServiceDisabled is a regression guard: when
// EnableEmailVerification is on but IsEmailServiceEnabled is off (e.g. no
// SMTP sender email configured), MagicLinkLogin used to still create a
// VerificationRequest row and fire an async SendEmail that was guaranteed
// to fail - the caller got a success message, but no email was ever
// deliverable and the token could never be verified. It must now behave
// the same as EnableEmailVerification being off entirely: no verification
// request created, matching signup.go's isEmailVerificationEnabled guard.
func TestMagicLinkLoginEmailServiceDisabled(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMagicLinkLogin = true
	cfg.EnableEmailVerification = true
	cfg.IsEmailServiceEnabled = false
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "magic_link_no_email_service_" + uuid.New().String() + "@authorizer.dev"

	res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
		Email: email,
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.Message)

	_, err = ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
	assert.Error(t, err, "no verification request should be created when the email service isn't configured")
}

// TestMagicLinkLoginMFAGate is a regression guard for VerifyEmail's MFA
// check: it used to be an ad-hoc TOTP-only condition
// (refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled &&
// isTOTPLoginEnabled) that silently skipped WebAuthn, email/SMS-OTP-as-MFA,
// EnforceMFA, and — most severely — MFALockedAt entirely, letting a locked
// or non-TOTP-MFA account complete a magic-link login with zero challenge.
// VerifyEmail now calls the same resolveMFAGate every other entry point
// (login.go, signup.go, oauth_mfa_gate.go, webauthn.go) uses.
func TestMagicLinkLoginMFAGate(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMagicLinkLogin = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.EnableMFA = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	startMagicLinkLogin := func(t *testing.T, email string) *schemas.VerificationRequest {
		t.Helper()
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
			Email: email,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		verificationRequest, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		require.NoError(t, err)
		require.NotNil(t, verificationRequest)
		return verificationRequest
	}

	t.Run("locked account is blocked, not logged in", func(t *testing.T) {
		email := "magic_link_locked_" + uuid.New().String() + "@authorizer.dev"
		verificationRequest := startMagicLinkLogin(t, email)

		// MagicLinkLogin creates the user record on the fly before the
		// verification email is sent - lock it before the click-through.
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		now := time.Now().Unix()
		user.MFALockedAt = &now
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: verificationRequest.Token,
		})
		assert.Error(t, err)
		assert.Nil(t, verifyRes)
		assert.Contains(t, err.Error(), "locked")
	})

	t.Run("MFA available and not yet configured withholds the token", func(t *testing.T) {
		email := "magic_link_offer_" + uuid.New().String() + "@authorizer.dev"
		verificationRequest := startMagicLinkLogin(t, email)

		verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: verificationRequest.Token,
		})
		require.NoError(t, err)
		require.NotNil(t, verifyRes)
		// The old ad-hoc check unconditionally issued a full session here
		// whenever EnableTOTPLogin was off (or the user had a
		// non-TOTP-only MFA config) - the gate must withhold instead.
		assert.Nil(t, verifyRes.AccessToken)
		assert.Equal(t, "Proceed to mfa setup", verifyRes.Message)
	})
}
