package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestVerifyEmailChallengesEmailOTPFactor is the regression guard for a bug
// found in a final whole-branch review, not by the earlier task-level audit
// of this file: service.VerifyEmail's authenticatorVerified was
// `totpVerified || hasWebauthnCredential`, completely omitting Email-OTP and
// SMS-OTP. A user whose only enrolled factor was Email-OTP or SMS-OTP, with
// HasSkippedMFASetupAt set (reachable: skip while unenrolled, then later
// enroll Email/SMS-OTP via settings without ever re-verifying TOTP/WebAuthn),
// resolved to mfaGateSkippedSetup and got a full token via magic-link login
// or signup email verification with zero MFA challenge - despite the exact
// same account being correctly challenged on a password login.
func TestVerifyEmailChallengesEmailOTPFactor(t *testing.T) {
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
	cfg.EnableEmailOTP = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_email_otp_factor_" + uuid.New().String() + "@authorizer.dev"
	res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{Email: email})
	require.NoError(t, err)
	require.NotNil(t, res)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	// The user previously skipped MFA setup, then later enrolled Email-OTP
	// via settings — both fields set, exactly the reachable combination that
	// triggered mfaGateSkippedSetup under the pre-fix authenticatorVerified.
	now := time.Now().Unix()
	user.HasSkippedMFASetupAt = &now
	user, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
		UserID:     user.ID,
		Method:     constants.EnvKeyEmailOTPAuthenticator,
		VerifiedAt: &now,
	})
	require.NoError(t, err)

	verificationRequest, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
	require.NoError(t, err)
	require.NotNil(t, verificationRequest)

	verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: verificationRequest.Token})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)
	assert.Nil(t, verifyRes.AccessToken, "must not issue a token before the enrolled Email-OTP factor is verified")
	assert.True(t, refs.BoolValue(verifyRes.ShouldShowEmailOtpScreen), "must challenge the account's enrolled Email-OTP factor")
}

// TestWebauthnLoginVerifyChallengesEmailOTPFactor is the same bug in
// WebauthnLoginVerify: authenticatorVerified only considered TOTP, so a
// passkey-primary login for a user whose only enrolled second factor was
// Email-OTP or SMS-OTP silently issued a token with zero MFA challenge.
func TestWebauthnLoginVerifyChallengesEmailOTPFactor(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableWebauthnMFA = true
	cfg.EnableEmailOTP = true
	cfg.IsEmailServiceEnabled = true
	ts := initTestSetup(t, cfg)

	// Per-user opt-in, not the global cfg.EnableMFA flag - signup itself must
	// stay unaffected so registerPasskeyForNewUser's SignUp call still issues
	// a token to register the passkey with. WebauthnLoginVerify's own gate
	// (below) only requires effectiveMFAEnabled, not cfg.EnableMFA globally.
	user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)

	now := time.Now().Unix()
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	user.HasSkippedMFASetupAt = &now
	user, err := ts.StorageProvider.UpdateUser(t.Context(), user)
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
		UserID:     user.ID,
		Method:     constants.EnvKeyEmailOTPAuthenticator,
		VerifiedAt: &now,
	})
	require.NoError(t, err)

	res, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Nil(t, res.AccessToken, "must not issue a token before the enrolled Email-OTP factor is verified")
	assert.True(t, refs.BoolValue(res.ShouldShowEmailOtpScreen), "must challenge the account's enrolled Email-OTP factor")
}
