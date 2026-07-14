package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestEmailOTPMFAEnrollment covers the full enroll-then-use cycle for
// email-OTP-as-MFA:
//   - a first-time, unenrolled login is withheld and offers email OTP setup
//     (mfaGateOfferAll's ShouldOfferEmailOtpMfaSetup, wired in login.go).
//   - EmailOTPMFASetup (bearer-token authenticated) creates an unverified
//     Authenticator row and does not by itself gate login.
//   - verify_otp marks that row verified.
//   - only after verification does a subsequent login route through the
//     retrofitted email-OTP-as-MFA branch instead of the offer-all screen.
//
// Sequencing note: EmailOTPMFASetup requires a bearer token, but the first
// login's token is withheld (mfaGateOfferAll). This test obtains a token via
// skip_mfa_setup first -- the realistic "skip now, add a second factor later
// from account settings" flow -- rather than calling EmailOTPMFASetup with no
// token, which cannot succeed.
func TestEmailOTPMFAEnrollment(t *testing.T) {
	const password = "Password@123"

	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	cfg.EnableEmailOTP = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	ts := initTestSetup(t, cfg)
	require.True(t, ts.Config.IsEmailServiceEnabled, "test SMTP fixture must derive IsEmailServiceEnabled=true")
	req, ctx := createContext(ts)

	email := "email_otp_mfa_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
	})
	require.NoError(t, err)
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	// First login: nothing enrolled yet -> withheld, offered.
	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken, "no method enrolled yet -> withheld, offer-all")
	assert.True(t, refs.BoolValue(loginRes.ShouldOfferEmailOtpMfaSetup))

	// Get a bearer token the realistic way: skip the offer now (settings-
	// screen enrollment is a separate, later, already-logged-in action).
	mfaSession := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaSession, "login must have set an mfa session cookie on the response")
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
	skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
	require.NoError(t, err)
	require.NotNil(t, skipRes.AccessToken)
	req.Header.Set("Authorization", "Bearer "+*skipRes.AccessToken)

	// Now, as an already-logged-in user, enroll a second factor.
	setupRes, err := ts.GraphQLProvider.EmailOTPMFASetup(ctx)
	require.NoError(t, err)
	require.NotNil(t, setupRes)

	authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")

	// Verify the code EmailOTPMFASetup sent. The test can't intercept the
	// outgoing email, so overwrite the stored digest with one for a known
	// plaintext, mirroring TestVerifyOTP's pattern.
	const knownPlainOTP = "654321"
	storedOTP, err := ts.StorageProvider.GetOTPByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, storedOTP)
	storedOTP.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
	storedOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, storedOTP)
	require.NoError(t, err)

	// verify_otp is identified by the mfa session cookie, not the bearer
	// token -- arm a fresh session directly (same approach as
	// TestVerifyOTPNoRecord) rather than threading the earlier one through.
	verifySession := uuid.NewString()
	require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, verifySession, time.Now().Add(5*time.Minute).Unix()))
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", verifySession))
	verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: &email, Otp: knownPlainOTP})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)

	authenticator, err = ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	require.NotNil(t, authenticator.VerifiedAt, "verify_otp must mark the pending enrollment verified")

	// A subsequent login now routes through the retrofitted email-OTP-as-MFA
	// challenge branch (not mfaGateOfferAll) since the enrollment is verified.
	req.Header.Set("Authorization", "")
	secondLogin, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.Nil(t, secondLogin.AccessToken, "an enrolled-and-verified email OTP factor must challenge, not skip, login")
	assert.True(t, refs.BoolValue(secondLogin.ShouldShowEmailOtpScreen), "must route through the OTP challenge branch, not the offer-all screen")
}

// TestVerifyOTPDoesNotAutoEnroll ensures verify_otp's new VerifiedAt-marking
// logic only fires when a pending (unverified) Authenticator row already
// exists -- i.e. only for a code sent by EmailOTPMFASetup/SMSOTPMFASetup. A
// routine login-time OTP (e.g. the pre-MFA email-verification challenge)
// that never went through setup must remain a no-op for enrollment purposes:
// it must not silently create/verify an Authenticator row.
func TestVerifyOTPDoesNotAutoEnroll(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableEmailOTP = true
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// A basic-auth user with an unverified email and no pending verification
	// request: login.go's (non-MFA) "email not verified -> send OTP" branch
	// fires for this, entirely independent of any MFA enrollment.
	email := "plain_otp_" + uuid.NewString() + "@authorizer.dev"
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef(email),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: "irrelevant"})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken)
	require.True(t, refs.BoolValue(loginRes.ShouldShowEmailOtpScreen))

	const knownPlainOTP = "111222"
	storedOTP, err := ts.StorageProvider.GetOTPByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, storedOTP)
	storedOTP.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
	storedOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, storedOTP)
	require.NoError(t, err)

	mfaSession := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaSession)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
	verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: &email, Otp: knownPlainOTP})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)

	authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	assert.Error(t, err, "no Authenticator row should exist for a plain login-time OTP that never went through setup")
	assert.Nil(t, authenticator)
}
