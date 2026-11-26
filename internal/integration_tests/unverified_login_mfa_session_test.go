package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// unverifiedLoginTestConfig enables every MFA method and both delivery
// channels, so each endpoint that acts on an MFA session is reachable.
func unverifiedLoginTestConfig() *config.Config {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	cfg.EnableWebauthnMFA = true
	cfg.EnableEmailOTP = true
	cfg.EnableSMSOTP = true
	cfg.IsEmailServiceEnabled = true
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	return cfg
}

// unverifiedLoginChannel describes one of the two login branches that send a
// verification OTP for an account whose email / phone is not yet verified.
type unverifiedLoginChannel struct {
	name        string
	email       *string
	phoneNumber *string
	otpMessage  string
}

func (c unverifiedLoginChannel) loginRequest(password string) *model.LoginRequest {
	return &model.LoginRequest{Email: c.email, PhoneNumber: c.phoneNumber, Password: password}
}

// newUnverifiedLoginChannels creates one unverified email user and one
// unverified phone user, both with the given password.
func newUnverifiedLoginChannels(ctx context.Context, t *testing.T, ts *testSetup, password string) []unverifiedLoginChannel {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	h := string(hash)

	email := "unverified_login_" + uuid.NewString() + "@authorizer.dev"
	_, err = ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef(email),
		Password:      &h,
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
		Roles:         "user",
	})
	require.NoError(t, err)

	phone := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	_, err = ts.StorageProvider.AddUser(ctx, &schemas.User{
		PhoneNumber:   refs.NewStringRef(phone),
		Password:      &h,
		SignupMethods: constants.AuthRecipeMethodMobileBasicAuth,
		Roles:         "user",
	})
	require.NoError(t, err)

	return []unverifiedLoginChannel{
		{name: "email", email: refs.NewStringRef(email), otpMessage: "Please check email inbox for the OTP"},
		{name: "phone", phoneNumber: refs.NewStringRef(phone), otpMessage: "Please check text message for the OTP"},
	}
}

func lookupChannelUser(ctx context.Context, t *testing.T, ts *testSetup, ch unverifiedLoginChannel) *schemas.User {
	t.Helper()
	var user *schemas.User
	var err error
	if ch.email != nil {
		user, err = ts.StorageProvider.GetUserByEmail(ctx, *ch.email)
	} else {
		user, err = ts.StorageProvider.GetUserByPhoneNumber(ctx, *ch.phoneNumber)
	}
	require.NoError(t, err)
	return user
}

// TestUnverifiedLoginMFASessionCannotBypassPassword: a login against an
// account whose email/phone is still unverified sends a verification OTP
// before the password is checked. The MFA session it arms may only be
// redeemed by verify_otp with that OTP; it must not be accepted as proof of a
// completed first factor by any endpoint, or a wrong password would be enough
// to obtain tokens or enroll a factor.
func TestUnverifiedLoginMFASessionCannotBypassPassword(t *testing.T) {
	const password = "RealPassword@123"
	const wrongPassword = "WrongPassword@123"

	cfg := unverifiedLoginTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	for i, name := range []string{"email", "phone"} {
		// Each subtest gets fresh accounts so a state change made by one
		// endpoint (lock, skip, enrollment) cannot mask another.
		var ch unverifiedLoginChannel
		freshAccount := func(t *testing.T) {
			t.Helper()
			ch = newUnverifiedLoginChannels(ctx, t, ts, password)[i]
		}
		// armWrongPassword logs in with a wrong password and attaches the
		// resulting MFA session cookie to req.
		armWrongPassword := func(t *testing.T) {
			t.Helper()
			req.Header.Set("Cookie", "")
			req.Header.Set("Authorization", "")
			res, err := ts.GraphQLProvider.Login(ctx, ch.loginRequest(wrongPassword))
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, ch.otpMessage, res.Message)
			assert.Nil(t, res.AccessToken)
			mfaSession := latestMfaSessionCookie(ts)
			require.NotEmpty(t, mfaSession)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
		}

		t.Run(name+"/skip_mfa_setup", func(t *testing.T) {
			freshAccount(t)
			armWrongPassword(t)
			res, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: ch.email, PhoneNumber: ch.phoneNumber})
			assert.Error(t, err)
			assert.True(t, res == nil || res.AccessToken == nil, "skip_mfa_setup must not issue tokens")

			// The identifier-less form resolves the account from the session.
			armWrongPassword(t)
			res, err = ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{})
			assert.Error(t, err)
			assert.True(t, res == nil || res.AccessToken == nil, "skip_mfa_setup must not issue tokens")
		})

		t.Run(name+"/lock_mfa", func(t *testing.T) {
			freshAccount(t)
			armWrongPassword(t)
			_, err := ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{Email: ch.email, PhoneNumber: ch.phoneNumber})
			assert.Error(t, err)
			armWrongPassword(t)
			_, err = ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{})
			assert.Error(t, err)
			assert.Nil(t, lookupChannelUser(ctx, t, ts, ch).MFALockedAt, "account must not be locked")
		})

		t.Run(name+"/otp_mfa_setup", func(t *testing.T) {
			freshAccount(t)
			setupReq := &model.OtpMfaSetupRequest{Email: ch.email, PhoneNumber: ch.phoneNumber}

			armWrongPassword(t)
			_, err := ts.GraphQLProvider.EmailOTPMFASetup(ctx, setupReq)
			assert.Error(t, err, "email_otp_mfa_setup")

			armWrongPassword(t)
			_, err = ts.GraphQLProvider.SMSOTPMFASetup(ctx, setupReq)
			assert.Error(t, err, "sms_otp_mfa_setup")

			armWrongPassword(t)
			totpRes, err := ts.GraphQLProvider.TOTPMFASetup(ctx, setupReq)
			assert.Error(t, err, "totp_mfa_setup")
			assert.True(t, totpRes == nil || totpRes.AuthenticatorSecret == nil, "totp_mfa_setup must not return a secret")

			armWrongPassword(t)
			_, err = ts.GraphQLProvider.TOTPMFASetup(ctx, &model.OtpMfaSetupRequest{})
			assert.Error(t, err, "totp_mfa_setup without identifier")

			user := lookupChannelUser(ctx, t, ts, ch)
			for _, method := range []string{constants.EnvKeyEmailOTPAuthenticator, constants.EnvKeySMSOTPAuthenticator, constants.EnvKeyTOTPAuthenticator} {
				a, _ := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, method)
				assert.Nil(t, a, "no %s authenticator may be created", method)
			}
		})

		t.Run(name+"/resend_otp_without_identifier", func(t *testing.T) {
			freshAccount(t)
			armWrongPassword(t)
			_, err := ts.GraphQLProvider.ResendOTP(ctx, &model.ResendOTPRequest{})
			assert.Error(t, err)
		})

		t.Run(name+"/webauthn_registration", func(t *testing.T) {
			freshAccount(t)
			armWrongPassword(t)
			_, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, ch.email, ch.phoneNumber)
			assert.Error(t, err, "webauthn_registration_options")

			armWrongPassword(t)
			_, err = ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil, nil)
			assert.Error(t, err, "webauthn_registration_options without identifier")

			// The session check must reject the caller before the credential
			// is even looked at.
			armWrongPassword(t)
			res, err := ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{
				Credential:  "{}",
				Email:       ch.email,
				PhoneNumber: ch.phoneNumber,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid session")
			assert.True(t, res == nil || res.AccessToken == nil)
		})

		t.Run(name+"/verify_otp_without_correct_otp", func(t *testing.T) {
			freshAccount(t)
			for _, otp := range []string{"", "000000"} {
				armWrongPassword(t)
				res, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: ch.email, PhoneNumber: ch.phoneNumber, Otp: otp})
				assert.Error(t, err)
				assert.True(t, res == nil || res.AccessToken == nil, "verify_otp(%q) must not issue tokens", otp)

				armWrongPassword(t)
				res, err = ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Otp: otp})
				assert.Error(t, err)
				assert.True(t, res == nil || res.AccessToken == nil, "verify_otp(%q) without identifier must not issue tokens", otp)

				armWrongPassword(t)
				res, err = ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: ch.email, PhoneNumber: ch.phoneNumber, Otp: otp, IsTotp: refs.NewBoolRef(true)})
				assert.Error(t, err)
				assert.True(t, res == nil || res.AccessToken == nil, "verify_otp(%q, totp) must not issue tokens", otp)
			}
			user := lookupChannelUser(ctx, t, ts, ch)
			assert.Nil(t, user.EmailVerifiedAt)
			assert.Nil(t, user.PhoneNumberVerifiedAt)
		})
	}
}

// TestUnverifiedLoginVerificationOTPCompletesLogin: the legitimate flow for an
// unverified account — login with the correct password, receive the OTP,
// verify_otp — still issues tokens and marks the email/phone verified.
func TestUnverifiedLoginVerificationOTPCompletesLogin(t *testing.T) {
	const password = "RealPassword@123"
	const knownPlainOTP = "482913"

	cfg := unverifiedLoginTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	for _, ch := range newUnverifiedLoginChannels(ctx, t, ts, password) {
		t.Run(ch.name, func(t *testing.T) {
			req.Header.Set("Cookie", "")
			res, err := ts.GraphQLProvider.Login(ctx, ch.loginRequest(password))
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, ch.otpMessage, res.Message)
			assert.Nil(t, res.AccessToken)
			mfaSession := latestMfaSessionCookie(ts)
			require.NotEmpty(t, mfaSession)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

			// The OTP is only delivered out of band; replace the stored
			// digest with one for a known plaintext.
			var otp *schemas.OTP
			if ch.email != nil {
				otp, err = ts.StorageProvider.GetOTPByEmail(ctx, *ch.email)
			} else {
				otp, err = ts.StorageProvider.GetOTPByPhoneNumber(ctx, *ch.phoneNumber)
			}
			require.NoError(t, err)
			otp.Otp = crypto.HashOTP(knownPlainOTP, cfg.EncryptionKey)
			otp.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
			_, err = ts.StorageProvider.UpsertOTP(ctx, otp)
			require.NoError(t, err)

			verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: ch.email, PhoneNumber: ch.phoneNumber, Otp: knownPlainOTP})
			require.NoError(t, err)
			require.NotNil(t, verifyRes)
			require.NotNil(t, verifyRes.AccessToken)
			assert.NotEmpty(t, *verifyRes.AccessToken)

			user := lookupChannelUser(ctx, t, ts, ch)
			if ch.email != nil {
				assert.NotNil(t, user.EmailVerifiedAt)
			} else {
				assert.NotNil(t, user.PhoneNumberVerifiedAt)
			}
		})
	}
}

// TestPhoneSignupVerificationSessionRequiresOTP: with phone verification on,
// signup sends an OTP and arms an MFA session for the new account. That
// session must only complete via verify_otp; skip_mfa_setup or a factor
// enrollment must not issue tokens for a phone number nobody has verified.
func TestPhoneSignupVerificationSessionRequiresOTP(t *testing.T) {
	const password = "Password@123"
	const knownPlainOTP = "305718"

	cfg := unverifiedLoginTestConfig()
	cfg.EnablePhoneVerification = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	phone := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{PhoneNumber: &phone, Password: password, ConfirmPassword: password})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Nil(t, res.AccessToken)
	require.True(t, refs.BoolValue(res.ShouldShowMobileOtpScreen))
	mfaSession := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaSession)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

	skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{PhoneNumber: &phone})
	assert.Error(t, err)
	assert.True(t, skipRes == nil || skipRes.AccessToken == nil, "skip_mfa_setup must not issue tokens")
	_, err = ts.GraphQLProvider.TOTPMFASetup(ctx, &model.OtpMfaSetupRequest{PhoneNumber: &phone})
	assert.Error(t, err, "totp_mfa_setup")
	_, err = ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil, &phone)
	assert.Error(t, err, "webauthn_registration_options")

	otp, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, phone)
	require.NoError(t, err)
	otp.Otp = crypto.HashOTP(knownPlainOTP, cfg.EncryptionKey)
	_, err = ts.StorageProvider.UpsertOTP(ctx, otp)
	require.NoError(t, err)
	verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{PhoneNumber: &phone, Otp: knownPlainOTP})
	require.NoError(t, err)
	require.NotNil(t, verifyRes.AccessToken)

	user, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, phone)
	require.NoError(t, err)
	assert.NotNil(t, user.PhoneNumberVerifiedAt)
}
