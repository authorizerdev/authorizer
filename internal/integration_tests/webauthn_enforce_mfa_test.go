package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// registerPasskeyForNewUser signs up a fresh verified user, registers one
// passkey via a simulated ceremony, and returns everything a login-time
// assertion needs. Mirrors the setup in webauthn_test.go's
// TestWebauthnPasskeyRegistrationAndLogin.
func registerPasskeyForNewUser(t *testing.T, ts *testSetup) (*schemas.User, virtualwebauthn.RelyingParty, virtualwebauthn.Authenticator, virtualwebauthn.Credential) {
	t.Helper()
	req, ctx := createContext(ts)
	rp := testRelyingParty(t, ts)
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	email := "enforce_mfa_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes.AccessToken)
	req.Header.Set("Authorization", "Bearer "+*signupRes.AccessToken)

	optRes, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil, nil)
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(optRes.Options)
	require.NoError(t, err)
	authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
		UserHandle: []byte(attOpts.UserID),
	})
	authenticator.AddCredential(credential)
	attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
	_, err = ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{Credential: attResp})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	req.Header.Del("Authorization")
	return user, rp, authenticator, credential
}

func assertPasskeyLogin(t *testing.T, ts *testSetup, rp virtualwebauthn.RelyingParty, authenticator virtualwebauthn.Authenticator, credential virtualwebauthn.Credential) (*model.AuthResponse, error) {
	t.Helper()
	_, ctx := createContext(ts)
	optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, nil)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
	require.NoError(t, err)
	assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
	return ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
}

// TestWebauthnLoginVerifyEnforceMFA locks in the decided policy: a successful
// passkey assertion (registered with UserVerification: Required — device +
// biometric bundled into one ceremony) satisfies the MFA requirement on its
// own, exactly like a verified TOTP/OTP code does in verify_otp. Every passkey
// login below therefore issues a token directly, with no TOTP re-challenge,
// regardless of EnforceMFA or any other enrolled factor. The one thing that
// must NOT change is the password path: password alone still never satisfies
// EnforceMFA (last subtest).
func TestWebauthnLoginVerifyEnforceMFA(t *testing.T) {
	t.Run("EnforceMFA=true, TOTP also enrolled — passkey login issues the token, no TOTP re-prompt", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)
		now := time.Now().Unix()
		_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
			UserID: user.ID, Method: constants.EnvKeyTOTPAuthenticator,
			Secret: "dummy-secret", VerifiedAt: &now,
		})
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken, "a verified passkey satisfies MFA on its own — no TOTP re-challenge")
		assert.NotEmpty(t, *authRes.AccessToken)
		assert.False(t, refs.BoolValue(authRes.ShouldShowTotpScreen), "must not re-demand TOTP after a successful passkey verify")
	})

	t.Run("EnforceMFA=true, no other factor enrolled — passkey login still issues the token", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken)
		assert.Nil(t, authRes.AuthenticatorSecret, "no enrollment payload — the passkey itself is the satisfying factor")
	})

	t.Run("EnforceMFA=true, TOTP disabled server-wide — passkey login issues the token (no longer refused)", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = false
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken, "passkey satisfies MFA even when TOTP is unavailable server-wide")
	})

	t.Run("EnforceMFA=false, optional MFA enabled — passkey login issues the token", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken)
	})

	t.Run("only email-OTP enrolled — passkey login issues the token, no email-OTP challenge", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableEmailOTP = true
		cfg.IsEmailServiceEnabled = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)
		now := time.Now().Unix()
		_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
			UserID: user.ID, Method: constants.EnvKeyEmailOTPAuthenticator, VerifiedAt: &now,
		})
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken, "a passkey login must satisfy MFA without an email-OTP challenge")
		assert.False(t, refs.BoolValue(authRes.ShouldShowEmailOtpScreen), "must not send an email OTP after a successful passkey verify")
	})

	t.Run("only SMS-OTP enrolled — passkey login issues the token, no SMS-OTP challenge", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableSMSOTP = true
		// IsSMSServiceEnabled is already true in the test config.
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)
		now := time.Now().Unix()
		_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
			UserID: user.ID, Method: constants.EnvKeySMSOTPAuthenticator, VerifiedAt: &now,
		})
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken, "a passkey login must satisfy MFA without an SMS-OTP challenge")
		assert.False(t, refs.BoolValue(authRes.ShouldShowMobileOtpScreen), "must not send an SMS OTP after a successful passkey verify")
	})

	t.Run("all three other factors enrolled simultaneously (TOTP + email-OTP + SMS-OTP) — passkey login issues the token, no challenge for any", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = true
		cfg.EnableEmailOTP = true
		cfg.IsEmailServiceEnabled = true
		cfg.EnableSMSOTP = true
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)
		now := time.Now().Unix()
		for _, method := range []string{constants.EnvKeyTOTPAuthenticator, constants.EnvKeyEmailOTPAuthenticator, constants.EnvKeySMSOTPAuthenticator} {
			_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
				UserID: user.ID, Method: method, Secret: "dummy-secret", VerifiedAt: &now,
			})
			require.NoError(t, err)
		}

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		require.NoError(t, err)
		require.NotNil(t, authRes)
		require.NotNil(t, authRes.AccessToken, "a passkey login must satisfy MFA even when every other factor is also enrolled")
		assert.False(t, refs.BoolValue(authRes.ShouldShowTotpScreen))
		assert.False(t, refs.BoolValue(authRes.ShouldShowEmailOtpScreen))
		assert.False(t, refs.BoolValue(authRes.ShouldShowMobileOtpScreen))
	})

	t.Run("revoked account — passkey login refused despite an otherwise valid assertion", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		now := time.Now().Unix()
		user.RevokedTimestamp = &now
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		assert.Error(t, err, "passkey login must be refused once the account is revoked")
		assert.Nil(t, authRes)
		if err != nil {
			assert.Contains(t, err.Error(), "revoked")
		}
	})

	t.Run("MFA-locked account — passkey login refused", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		now := time.Now().Unix()
		user.MFALockedAt = &now
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)

		authRes, err := assertPasskeyLogin(t, ts, rp, authenticator, credential)
		assert.Error(t, err, "passkey login must be refused while the account's MFA is locked — a passkey satisfying MFA does not bypass an explicit lock")
		assert.Nil(t, authRes)
		if err != nil {
			assert.Contains(t, err.Error(), "locked")
		}
	})

	t.Run("password-only login without a second factor is STILL blocked by EnforceMFA (unchanged)", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnforceMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "enforce_pw_" + uuid.NewString() + "@authorizer.dev"
		password := "Password@123"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)
		now := time.Now().Unix()
		_, err = ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			Password:                 refs.NewStringRef(string(hash)),
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, loginRes)
		assert.Nil(t, loginRes.AccessToken, "password alone must not satisfy EnforceMFA — the passkey policy change must not weaken the password path")
	})
}

// TestWebauthnLoginVerifyAsSecondFactor is the original Bug 1 scenario: password
// is the primary factor and a passkey is offered as the second factor
// (ShouldOfferWebauthnMfaVerify). Completing that passkey must finish the login
// without any spurious TOTP prompt, even when TOTP is also enrolled.
func TestWebauthnLoginVerifyAsSecondFactor(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableWebauthnMFA = true
	cfg.EnableTOTPLogin = true
	// EnableMFA is left off during setup so signup still issues the bearer token
	// registerPasskeyForNewUser needs to enroll the passkey; it is turned on
	// (mutating the same cfg the providers read live) just before the login that
	// must exercise the password MFA gate.
	ts := initTestSetup(t, cfg)

	user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
	require.NoError(t, err)
	now := time.Now().Unix()
	_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
		UserID: user.ID, Method: constants.EnvKeyTOTPAuthenticator,
		Secret: "dummy-secret", VerifiedAt: &now,
	})
	require.NoError(t, err)

	cfg.EnableMFA = true

	email := refs.StringValue(user.Email)
	req, ctx := createContext(ts)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: "Password@123"})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken, "password login with a verified second factor must withhold the token and challenge it")
	require.True(t, refs.BoolValue(loginRes.ShouldOfferWebauthnMfaVerify), "the enrolled passkey must be offered as a verify method")

	mfaSession := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaSession, "password login must have armed an mfa session cookie")
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

	optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &email)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
	require.NoError(t, err)
	assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)

	verifyRes, err := ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)
	require.NotNil(t, verifyRes.AccessToken, "completing the offered passkey second factor must finish the login")
	assert.NotEmpty(t, *verifyRes.AccessToken)
	assert.False(t, refs.BoolValue(verifyRes.ShouldShowTotpScreen), "must not re-demand TOTP after the passkey second factor")
}

// TestWebauthnLoginVerifyAsSecondFactorEmailOTPEnrolled guards the fix for a
// gap this test caught: the email-OTP branch in login.go used to return early
// — before a registered passkey was ever checked for — forcing a user with
// both an enrolled passkey and email-OTP into the email-OTP screen with no
// passkey alternative offered. login.go now computes hasWebauthnCredential
// up-front and offers it alongside email-OTP, matching how TOTP is already
// offered alongside webauthn.
func TestWebauthnLoginVerifyAsSecondFactorEmailOTPEnrolled(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableWebauthnMFA = true
	cfg.EnableEmailOTP = true
	cfg.IsEmailServiceEnabled = true
	ts := initTestSetup(t, cfg)

	user, _, _, _ := registerPasskeyForNewUser(t, ts)
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
	require.NoError(t, err)
	now := time.Now().Unix()
	_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
		UserID: user.ID, Method: constants.EnvKeyEmailOTPAuthenticator, VerifiedAt: &now,
	})
	require.NoError(t, err)

	cfg.EnableMFA = true

	email := refs.StringValue(user.Email)
	_, ctx := createContext(ts)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: "Password@123"})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken)
	assert.True(t, refs.BoolValue(loginRes.ShouldShowEmailOtpScreen), "email-OTP is still challenged")
	assert.True(t, refs.BoolValue(loginRes.ShouldOfferWebauthnMfaVerify), "the enrolled passkey must be offered as an alternative to email-OTP")
}

// TestWebauthnLoginVerifyAsSecondFactorSMSOTPEnrolled mirrors the email-OTP
// test above for the SMS-OTP branch.
func TestWebauthnLoginVerifyAsSecondFactorSMSOTPEnrolled(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableWebauthnMFA = true
	cfg.EnableSMSOTP = true
	ts := initTestSetup(t, cfg)

	user, _, _, _ := registerPasskeyForNewUser(t, ts)
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
	require.NoError(t, err)
	now := time.Now().Unix()
	_, err = ts.StorageProvider.AddAuthenticator(t.Context(), &schemas.Authenticator{
		UserID: user.ID, Method: constants.EnvKeySMSOTPAuthenticator, VerifiedAt: &now,
	})
	require.NoError(t, err)

	cfg.EnableMFA = true

	email := refs.StringValue(user.Email)
	_, ctx := createContext(ts)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: "Password@123"})
	require.NoError(t, err)
	require.Nil(t, loginRes.AccessToken)
	assert.True(t, refs.BoolValue(loginRes.ShouldShowMobileOtpScreen), "SMS-OTP is still challenged")
	assert.True(t, refs.BoolValue(loginRes.ShouldOfferWebauthnMfaVerify), "the enrolled passkey must be offered as an alternative to SMS-OTP")
}

// TestWebauthnLoginOptionsRejectsPasswordResetSession is the regression test
// for the session-purpose gap in WebauthnLoginOptions' scoped (MFA-alternative)
// flow: a password_reset-purpose session (minted only by ForgotPassword) must
// not be redeemable here -- the returned PublicKeyCredentialRequestOptions,
// including the account's own credential IDs, is itself the leak this gate
// exists to prevent (see the comment on WebauthnLoginOptions). Verified and
// Challenge sessions -- the same two VerifyOTP accepts for the equivalent
// TOTP-alternative flow -- must still work.
func TestWebauthnLoginOptionsRejectsPasswordResetSession(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableWebauthnMFA = true
	ts := initTestSetup(t, cfg)

	user, _, _, _ := registerPasskeyForNewUser(t, ts)
	email := refs.StringValue(user.Email)
	req, ctx := createContext(ts)

	t.Run("a password_reset session is rejected", func(t *testing.T) {
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposePasswordReset, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		_, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &email)
		assert.Error(t, err, "a password_reset session must not return this account's passkey login options")
	})

	t.Run("a Verified session is still accepted", func(t *testing.T) {
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		_, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &email)
		assert.NoError(t, err)
	})

	t.Run("a Challenge session is still accepted", func(t *testing.T) {
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		_, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &email)
		assert.NoError(t, err)
	})
}
