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
	setupRes, err := ts.GraphQLProvider.EmailOTPMFASetup(ctx, nil)
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
	require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, verifySession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
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

// TestEmailOTPMFASetupViaMfaSessionCookie proves the actual gap this fix
// closes: a user offered "set up Email OTP" from the token-withheld
// mfaGateOfferAll screen has NO bearer token, only the MFA session cookie —
// so email_otp_mfa_setup must be reachable in that cookie-only mode, not
// just from an already-logged-in settings screen. Full flow: withheld
// first login -> email_otp_mfa_setup via cookie+email (no Authorization
// header at all) -> verify_otp -> the previously-withheld token is issued.
func TestEmailOTPMFASetupViaMfaSessionCookie(t *testing.T) {
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
	req, ctx := createContext(ts)

	email := "cookie_email_otp_mfa_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
	})
	require.NoError(t, err)
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	// First login: nothing enrolled yet -> withheld, offered, no bearer
	// token issued.
	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken, "no method enrolled yet -> withheld, offer-all")
	assert.True(t, refs.BoolValue(loginRes.ShouldOfferEmailOtpMfaSetup))

	mfaSession := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaSession, "login must have set an mfa session cookie on the response")
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
	// Deliberately no Authorization header — this is the whole point: the
	// caller has no bearer token yet, only the cookie + email.
	req.Header.Set("Authorization", "")

	setupRes, err := ts.GraphQLProvider.EmailOTPMFASetup(ctx, &model.OtpMfaSetupRequest{Email: &email})
	require.NoError(t, err, "email_otp_mfa_setup must be reachable via cookie+email with no bearer token")
	require.NotNil(t, setupRes)

	authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")

	const knownPlainOTP = "482913"
	storedOTP, err := ts.StorageProvider.GetOTPByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, storedOTP)
	storedOTP.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
	storedOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, storedOTP)
	require.NoError(t, err)

	// Same MFA session cookie is still valid -- verify_otp completes the
	// still-in-progress, still-withheld login.
	verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: &email, Otp: knownPlainOTP})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)
	require.NotNil(t, verifyRes.AccessToken, "verify_otp must issue the token that was withheld at login")

	authenticator, err = ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	require.NotNil(t, authenticator.VerifiedAt, "verify_otp must mark the pending enrollment verified")
}

// TestSMSOTPMFASetupViaMfaSessionCookie is TestEmailOTPMFASetupViaMfaSessionCookie's
// SMS twin -- confirms sms_otp_mfa_setup is reachable the same cookie-only
// way, keyed by phone_number instead of email, and that the full chain
// (withheld login -> cookie-authenticated setup -> verify_otp -> the
// withheld token being issued) closes the same as the email-OTP twin.
func TestSMSOTPMFASetupViaMfaSessionCookie(t *testing.T) {
	const password = "Password@123"

	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableSMSOTP = true
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	cfg.TwilioAPISecret = "test-twilio-api-secret"
	cfg.TwilioAPIKey = "test-twilio-api-key"
	cfg.TwilioAccountSID = "test-twilio-account-sid"
	cfg.TwilioSender = "test-twilio-sender"
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		PhoneNumber: &mobile, Password: password, ConfirmPassword: password,
		IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
	})
	require.NoError(t, err)
	user, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	// Signup's own verification OTP is irrelevant to this test; mark the
	// phone verified directly so login reaches the MFA gate instead of the
	// phone-verification challenge.
	now := time.Now().Unix()
	user.PhoneNumberVerifiedAt = &now
	_, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{PhoneNumber: &mobile, Password: password})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken, "no method enrolled yet -> withheld, offer-all")
	assert.True(t, refs.BoolValue(loginRes.ShouldOfferSmsOtpMfaSetup))

	mfaSession := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaSession)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
	req.Header.Set("Authorization", "")

	setupRes, err := ts.GraphQLProvider.SMSOTPMFASetup(ctx, &model.OtpMfaSetupRequest{PhoneNumber: &mobile})
	require.NoError(t, err, "sms_otp_mfa_setup must be reachable via cookie+phone_number with no bearer token")
	require.NotNil(t, setupRes)

	authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	assert.Nil(t, authenticator.VerifiedAt, "setup alone must not mark the enrollment verified")

	// Complete the chain: the test can't intercept the outgoing SMS, so
	// overwrite the stored digest with one for a known plaintext, same as
	// the email-OTP twin.
	const knownPlainOTP = "739104"
	storedOTP, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	require.NotNil(t, storedOTP)
	storedOTP.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
	storedOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, storedOTP)
	require.NoError(t, err)

	// Same MFA session cookie is still valid -- verify_otp completes the
	// still-in-progress, still-withheld login.
	verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{PhoneNumber: &mobile, Otp: knownPlainOTP})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)
	require.NotNil(t, verifyRes.AccessToken, "verify_otp must issue the token that was withheld at login")

	authenticator, err = ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
	require.NoError(t, err)
	require.NotNil(t, authenticator)
	require.NotNil(t, authenticator.VerifiedAt, "verify_otp must mark the pending enrollment verified")
}

// TestOTPMFASetupRejectsUnauthenticatedCaller confirms the new dual-mode
// resolution fails closed: a caller with neither a valid bearer token/session
// NOR a valid MFA session cookie + email/phone_number must be rejected, for
// both email_otp_mfa_setup and sms_otp_mfa_setup.
func TestOTPMFASetupRejectsUnauthenticatedCaller(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableEmailOTP = true
	cfg.IsEmailServiceEnabled = true
	cfg.EnableSMSOTP = true
	cfg.IsSMSServiceEnabled = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// No Authorization header, no MFA session cookie, no params at all.
	_, err := ts.GraphQLProvider.EmailOTPMFASetup(ctx, nil)
	assert.Error(t, err, "no token and no cookie/identity must be rejected")

	_, err = ts.GraphQLProvider.SMSOTPMFASetup(ctx, nil)
	assert.Error(t, err, "no token and no cookie/identity must be rejected")

	// email/phone_number supplied but no MFA session cookie set at all --
	// still must be rejected (params alone never authenticate).
	someEmail := "no_cookie_" + uuid.NewString() + "@authorizer.dev"
	_, err = ts.GraphQLProvider.EmailOTPMFASetup(ctx, &model.OtpMfaSetupRequest{Email: &someEmail})
	assert.Error(t, err, "email/phone_number without a valid mfa session cookie must be rejected")
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
