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

// TestLoginMFACrossIdentifierChallenge is the regression guard for the bug
// where login.go's inline email/SMS-OTP MFA challenge only fired when the
// enrolled method matched the identifier the caller logged in with. A user
// who signed up with email, later verified a phone number, and enrolled
// SMS-OTP as their second factor was silently NOT challenged on an
// email+password login — the SMS branch required isMobileLogin — and fell
// through to resolveMFAGate, which (correctly) does not count email/SMS OTP,
// so they were offered a fresh setup instead of being blocked to verify the
// factor they already opted into.
//
// The challenge must now fire on enrollment alone and send the code to the
// account's own stored contact (user.PhoneNumber), independent of the login
// identifier.
func TestLoginMFACrossIdentifierChallenge(t *testing.T) {
	const password = "Password@123"

	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableSMSOTP = true
	cfg.IsSMSServiceEnabled = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Email signup (auto-verified: email verification is off in getTestConfig)
	// so the stored password hash is one login.go's bcrypt check accepts.
	email := "login_cross_id_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	// Later this account verifies a phone number and opts into MFA.
	now := time.Now().Unix()
	phone := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	user.PhoneNumber = refs.NewStringRef(phone)
	user.PhoneNumberVerifiedAt = &now
	user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	user, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)

	// SMS-OTP is the user's ONLY enrolled/verified second factor.
	_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
		UserID:     user.ID,
		Method:     constants.EnvKeySMSOTPAuthenticator,
		VerifiedAt: &now,
	})
	require.NoError(t, err)

	// Login with EMAIL + password (not phone). The SMS-OTP factor must still
	// be challenged.
	res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Nil(t, res.AccessToken, "must not issue a token before the enrolled SMS-OTP factor is verified")
	assert.True(t, refs.BoolValue(res.ShouldShowMobileOtpScreen), "an email login must still challenge the account's enrolled SMS-OTP factor")
	assert.False(t, refs.BoolValue(res.ShouldOfferSmsOtpMfaSetup), "the user already enrolled SMS-OTP; this must be a verify challenge, not a setup offer")

	// The plaintext OTP is only sent over SMS (which the suite can't
	// intercept) and stored as an HMAC digest, keyed by both email and phone
	// (generateAndStoreOTP writes both). Overwrite it with a known
	// plaintext/digest pair, then complete the challenge via the phone.
	storedOTP, err := ts.StorageProvider.GetOTPByPhoneNumber(ctx, phone)
	require.NoError(t, err)
	require.NotNil(t, storedOTP)
	const knownPlainOTP = "123456"
	storedOTP.Otp = crypto.HashOTP(knownPlainOTP, cfg.JWTSecret)
	storedOTP.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	_, err = ts.StorageProvider.UpsertOTP(ctx, storedOTP)
	require.NoError(t, err)

	// The MFA session cookie is only set on the login response; copy it onto
	// the next request by hand (http.Request cookies aren't auto-updated from
	// responses in this in-process setup).
	mfaCookie := latestMfaSessionCookie(ts)
	require.NotEmpty(t, mfaCookie, "the SMS-OTP challenge must arm an mfa session cookie")
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaCookie))

	verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
		PhoneNumber: &phone,
		Otp:         knownPlainOTP,
	})
	require.NoError(t, err)
	require.NotNil(t, verifyRes)
	assert.NotNil(t, verifyRes.AccessToken, "verifying the SMS OTP must complete login")
	assert.NotEmpty(t, *verifyRes.AccessToken)
}
