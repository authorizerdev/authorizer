package integration_tests

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Audit finding #3: verification-token purpose confusion.
//
// Magic-link, signup-verification, invite and forgot-password tokens all live
// in one `verification_requests` table keyed by the token string, and
// GetVerificationRequestByToken matches on the token alone. Neither consumer
// used to check what the token was actually minted for, so any leaked
// verification link (referer leakage, proxy/access logs, a shared URL) could be
// POSTed to the wrong endpoint:
//
//   - a magic-link token redeemed at ResetPassword sets an attacker-chosen
//     password AND appends basic_auth to the account's signup methods,
//     escalating a one-shot passwordless capability into durable account
//     takeover;
//   - a forgot-password token redeemed at VerifyEmail hands out a full session.
//
// Both directions are pinned below, along with the happy paths, so the guard
// cannot be tightened into breaking legitimate flows.

func TestVerificationTokenPurposeBinding(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.EnableMagicLinkLogin = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("a magic-link token cannot be redeemed at ResetPassword", func(t *testing.T) {
		email := "purpose_magic_" + uuid.NewString() + "@authorizer.dev"

		// Mint a real magic-link token through the real flow.
		_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{Email: email})
		require.NoError(t, err)
		vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		require.NoError(t, err)
		require.NotEmpty(t, vr.Token)

		res, err := ts.GraphQLProvider.ResetPassword(ctx, &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(vr.Token),
			Password:        "AttackerChosen@123",
			ConfirmPassword: "AttackerChosen@123",
		})
		require.Error(t, err, "a magic-link token must not be redeemable for a password change")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid token")

		// The account must be untouched: no password set, no basic_auth added
		// to signup methods (that append is what makes this an ATO rather than
		// a nuisance).
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		if user.Password != nil {
			assert.NotEqual(t, nil, bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte("AttackerChosen@123")),
				"the attacker's password must not have been set")
		}
		assert.NotContains(t, user.SignupMethods, constants.AuthRecipeMethodBasicAuth,
			"a refused reset must not add basic_auth to the account")

		// The token itself must still be intact for its real purpose.
		still, err := ts.StorageProvider.GetVerificationRequestByToken(ctx, vr.Token)
		require.NoError(t, err)
		assert.Equal(t, constants.VerificationTypeMagicLinkLogin, still.Identifier)
	})

	t.Run("a signup-verification token cannot be redeemed at ResetPassword", func(t *testing.T) {
		email := "purpose_signup_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        "Password@123",
			ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)
		vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.ResetPassword(ctx, &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(vr.Token),
			Password:        "AttackerChosen@123",
			ConfirmPassword: "AttackerChosen@123",
		})
		require.Error(t, err, "a signup token must not be redeemable for a password change")
		assert.Nil(t, res)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, user.Password)
		assert.Error(t, bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte("AttackerChosen@123")),
			"the attacker's password must not have been set")
	})

	t.Run("a forgot-password token cannot be redeemed at VerifyEmail", func(t *testing.T) {
		email := "purpose_forgot_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        "Password@123",
			ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)
		// Consume the signup verification so the account is live.
		signupVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: signupVR.Token})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{Email: &email})
		require.NoError(t, err)
		forgotVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: forgotVR.Token})
		require.Error(t, err, "a forgot-password token must not be redeemable for a session")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid verification token")
	})

	t.Run("each token still works for the purpose it was minted for", func(t *testing.T) {
		// Signup token -> VerifyEmail.
		email := "purpose_happy_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        "Password@123",
			ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)
		signupVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		verified, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: signupVR.Token})
		require.NoError(t, err, "the signup token must still complete signup")
		require.NotNil(t, verified)

		// Forgot-password token -> ResetPassword.
		_, err = ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{Email: &email})
		require.NoError(t, err)
		forgotVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)
		reset, err := ts.GraphQLProvider.ResetPassword(ctx, &model.ResetPasswordRequest{
			Token:           refs.NewStringRef(forgotVR.Token),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		})
		require.NoError(t, err, "the forgot-password token must still reset the password")
		require.NotNil(t, reset)

		// Magic-link token -> VerifyEmail (the magic-link family IS served here).
		magicEmail := "purpose_happy_magic_" + uuid.NewString() + "@authorizer.dev"
		_, err = ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{Email: magicEmail})
		require.NoError(t, err)
		magicVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, magicEmail, constants.VerificationTypeMagicLinkLogin)
		require.NoError(t, err)
		magicRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: magicVR.Token})
		require.NoError(t, err, "a magic-link token must still complete a magic-link login")
		require.NotNil(t, magicRes)
	})
}

// TestVerificationTokenPurposeBindingREST pins the same guard on GET
// /verify_email.
//
// That route is a SEPARATE implementation from the GraphQL mutation — it does
// its own GetVerificationRequestByToken / ParseJWTToken / ValidateJWTClaims —
// and it is the URL every verification and magic-link mail actually points at
// (utils.GetEmailVerificationURL). Gating only the mutation left the route that
// receives real traffic wide open: a forgot-password token redeemed here issued
// a full session AND marked the address verified. The same split already caused
// the MFA gate to be missed on this handler once (see
// TestVerifyEmailRESTEndpointMFAGate), so both directions are pinned here.
func TestVerificationTokenPurposeBindingREST(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	hitVerifyEmail := func(t *testing.T, token string) *http.Response {
		t.Helper()
		resp, err := httpClient.Get(ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(token) +
			"&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback"))
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	email := "purpose_rest_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)
	signupVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	// The handler reports both success and failure as a 307 to redirect_uri;
	// what separates them is whether the Location carries tokens or an error.
	t.Run("a signup token still completes verification here", func(t *testing.T) {
		resp := hitVerifyEmail(t, signupVR.Token)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Location"), "access_token=",
			"the purpose this route serves must keep working")

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, user.EmailVerifiedAt)
	})

	t.Run("a forgot-password token is refused", func(t *testing.T) {
		_, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{Email: &email})
		require.NoError(t, err)
		forgotVR, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)

		resp := hitVerifyEmail(t, forgotVR.Token)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		location := resp.Header.Get("Location")
		assert.NotContains(t, location, "access_token=",
			"a password-reset token must not be redeemable for a session")
		assert.Contains(t, location, "error=")

		// The token must survive: refusing it here cannot consume the reset the
		// rightful owner is still holding.
		still, err := ts.StorageProvider.GetVerificationRequestByToken(ctx, forgotVR.Token)
		require.NoError(t, err, "the forgot-password request must not have been deleted")
		assert.Equal(t, constants.VerificationTypeForgotPassword, still.Identifier)
	})
}

// TestResetPasswordVerifiesEmail pins the self-service recovery path.
//
// A forgot-password token is emailed to the address, is single-use and
// nonce-bound, so completing the reset proves control of that mailbox and must
// mark the address verified. This used to only happen for accounts that did not
// already have basic_auth among their signup methods — so an unverified
// password account could complete a reset and stay unverified forever, with no
// self-service way out.
//
// That matters because an unverified account now blocks a federated login for
// the same address (account pre-hijacking defense): forgot-password is the one
// recovery the rightful mailbox owner can drive without an admin.
func TestResetPasswordVerifiesEmail(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "reset_verifies_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)

	// The account exists but nobody has proven control of the address yet —
	// exactly the state that blocks a social login for the same email.
	before, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.Nil(t, before.EmailVerifiedAt, "signup with verification enabled leaves the address unverified")
	require.Contains(t, before.SignupMethods, constants.AuthRecipeMethodBasicAuth)

	_, err = ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{Email: &email})
	require.NoError(t, err)
	vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
	require.NoError(t, err)

	_, err = ts.GraphQLProvider.ResetPassword(ctx, &model.ResetPasswordRequest{
		Token:           refs.NewStringRef(vr.Token),
		Password:        "NewPassword@123",
		ConfirmPassword: "NewPassword@123",
	})
	require.NoError(t, err)

	after, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, after.EmailVerifiedAt,
		"receiving and redeeming the emailed token proves mailbox control, so the address is now verified")
	assert.Equal(t, before.ID, after.ID, "recovery must not replace the account")
}

// TestResendVerifyEmailMintsFreshRequest pins the primary self-service recovery.
//
// Signup already mails a verification link, and an expired verification row is
// still returned by GetVerificationRequestByEmail (no expiry filter), so the
// ordinary expired-link case always worked. The gap was narrower: a password
// login attempt PURGES the expired row (login.go), and after that
// resend_verify_email found nothing and silently did nothing — leaving the user
// with no way to ask for a new link, and therefore no way to verify at all.
func TestResendVerifyEmailMintsFreshRequest(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "resend_fresh_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)

	// Signup mails a link and records the request.
	original, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err, "signup must create a verification request")

	// Simulate the purge a post-expiry password login performs.
	require.NoError(t, ts.StorageProvider.DeleteVerificationRequest(ctx, original))
	_, err = ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.Error(t, err, "no pending verification request remains")

	// The user asks for a new link. This used to silently do nothing.
	_, err = ts.GraphQLProvider.ResendVerifyEmail(ctx, &model.ResendVerifyEmailRequest{
		Email:      email,
		Identifier: constants.VerificationTypeBasicAuthSignup,
	})
	require.NoError(t, err)

	fresh, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err, "a fresh verification request must be minted when none is pending")
	assert.NotEqual(t, original.Token, fresh.Token, "the new link must be a new token")

	// And it actually verifies the address end to end.
	_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: fresh.Token})
	require.NoError(t, err, "the resent link must complete verification")

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, user.EmailVerifiedAt)
}

// TestResendVerifyEmailIsNotAnOpenMailer guards the other side of that change:
// minting on demand must not let anyone who knows a registered address use this
// endpoint to send them mail.
func TestResendVerifyEmailIsNotAnOpenMailer(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "resend_mailer_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)
	vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)
	_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: vr.Token})
	require.NoError(t, err)

	// Address is now verified — there is nothing left to verify.
	_, err = ts.GraphQLProvider.ResendVerifyEmail(ctx, &model.ResendVerifyEmailRequest{
		Email:      email,
		Identifier: constants.VerificationTypeBasicAuthSignup,
	})
	require.NoError(t, err, "the response stays generic so it is not an existence oracle")

	_, err = ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	assert.Error(t, err, "no verification request may be minted for an already-verified address")
}

// TestVerifyEmailMarksVerifiedBeforeMFAGate is the regression guard for a user
// who clicks their verification link, is sent to the MFA setup screen, and is
// then told forever that their email is not verified.
//
// MFA is on by default (TOTP needs no external provider, so config.Finalize
// derives EnableMFA=true), so a fresh signup's verification click lands on
// resolveMFAGate's offer/enroll branch — which returns EARLY. The
// email_verified_at write used to sit after that return, so it never happened.
// The account then failed every later check that gates on it; passkey login in
// particular refuses with "email is not verified. please verify your email
// before signing in with a passkey", which the user cannot resolve by verifying
// again.
func TestVerifyEmailMarksVerifiedBeforeMFAGate(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	// Left at the default (derived true) on purpose — this is the configuration
	// that triggers the bug.
	cfg.EnableMFA = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_before_mfa_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)

	before, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.Nil(t, before.EmailVerifiedAt)

	vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	// The response may be a session OR an MFA setup screen depending on the
	// gate — this test deliberately does not care which. What matters is that
	// the address is recorded as verified either way.
	_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: vr.Token})
	require.NoError(t, err)

	after, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, after.EmailVerifiedAt,
		"clicking the verification link must record the address as verified even when MFA interrupts the login")
	assert.Equal(t, before.ID, after.ID)
}

// TestSignupDoesNotLeakAccountExistence guards the account-existence oracle:
// signing up with an address that is already registered must be
// indistinguishable from signing up with a fresh one.
//
// Timing was already equalised with a dummy bcrypt, but the RESPONSE differed —
// a distinct "signup failed" for a taken address versus "check your inbox" for
// a free one is enough to enumerate which addresses hold accounts, which is how
// targeted phishing and credential-stuffing lists get built.
func TestSignupDoesNotLeakAccountExistence(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	taken := "signup_oracle_" + uuid.NewString() + "@authorizer.dev"
	fresh := "signup_oracle_" + uuid.NewString() + "@authorizer.dev"

	signup := func(email string) (*model.AuthResponse, error) {
		return ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        "Password@123",
			ConfirmPassword: "Password@123",
		})
	}

	first, err := signup(taken)
	require.NoError(t, err)
	require.NotNil(t, first)

	// Same address again — the probe an attacker actually runs.
	collision, collisionErr := signup(taken)
	// A different fresh address — the control.
	control, controlErr := signup(fresh)

	require.NoError(t, controlErr)
	require.NotNil(t, control)

	assert.NoError(t, collisionErr,
		"a taken address must not answer with an error while a free one succeeds")
	require.NotNil(t, collision)
	assert.Equal(t, control.Message, collision.Message,
		"the two responses must be byte-identical or the address is enumerable")
	assert.Nil(t, collision.AccessToken, "a collision must never hand out a session")
	assert.Nil(t, collision.User, "a collision must not echo back the existing account")

	// And the guard must not have created a second account for that address.
	original, err := ts.StorageProvider.GetUserByEmail(ctx, taken)
	require.NoError(t, err)
	assert.NotEmpty(t, original.ID, "the original account is still the one on file")
}
