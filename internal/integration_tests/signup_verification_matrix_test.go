package integration_tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Signup × email-verification matrix.
//
// The audit work touched every one of these paths, and one of them shipped a
// bug nobody caught until a user hit it: a verification click that landed on
// the MFA gate returned without ever writing email_verified_at, so the account
// was permanently rejected by anything gating on it (passkey login most
// visibly). That bug existed because each flow was tested in isolation and the
// INVARIANT across them was not.
//
// The invariant these tests pin: whenever a principal has proven control of
// their address — by clicking a mailed link, by redeeming a mailed OTP, or by
// the operator disabling verification entirely — the account ends up with
// email_verified_at set. Anything else leaves users in a state they cannot
// escape on their own.

// verificationState is the observable outcome of a signup, independent of which
// screen the API happened to return.
type verificationState struct {
	exists   bool
	verified bool
	methods  string
}

func readVerificationState(t *testing.T, ts *testSetup, ctx context.Context, email string) verificationState {
	t.Helper()
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return verificationState{}
	}
	return verificationState{
		exists:   true,
		verified: user.EmailVerifiedAt != nil,
		methods:  user.SignupMethods,
	}
}

// TestSignupVerificationMatrix walks the email-signup paths under both
// verification settings and both MFA settings, and asserts the same invariant
// in every cell.
func TestSignupVerificationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		verificationOn        bool
		mfaOn                 bool
		expectSessionAtSignup bool
	}{
		{"verification off, mfa off", false, false, true},
		{"verification off, mfa on", false, true, true},
		{"verification on, mfa off", true, false, false},
		// The cell that produced the reported bug: the verification click lands
		// on the MFA gate, which returns before the old code wrote
		// email_verified_at.
		{"verification on, mfa on", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := getTestConfig()
			cfg.IsEmailServiceEnabled = true
			cfg.EnableEmailVerification = tc.verificationOn
			cfg.DisableMFA = !tc.mfaOn
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := fmt.Sprintf("matrix_%s_%s@authorizer.dev", uuid.NewString(), "x")
			res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        "Password@123",
				ConfirmPassword: "Password@123",
			})
			require.NoError(t, err)
			require.NotNil(t, res)

			state := readVerificationState(t, ts, ctx, email)
			require.True(t, state.exists, "signup must create the account in every cell")
			assert.Contains(t, state.methods, constants.AuthRecipeMethodBasicAuth)

			if !tc.verificationOn {
				// Nobody is going to prove anything, so the operator has said
				// the address counts as good on arrival.
				assert.True(t, state.verified,
					"with verification disabled the address is verified at signup, or the user can never become verified at all")
				assert.NotNil(t, res.AccessToken, "signup issues a session when there is nothing to verify")
				return
			}

			// Verification on: no session yet, and a pending request exists.
			assert.False(t, state.verified, "unverified until the link is clicked")
			assert.Nil(t, res.AccessToken, "an unverified signup must not hand out a session")

			vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
			require.NoError(t, err, "a verification request must exist to be clickable")

			// Click it. The response may be a session OR an MFA screen — the
			// invariant is about the stored state, not the screen.
			_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: vr.Token})
			require.NoError(t, err)

			after := readVerificationState(t, ts, ctx, email)
			assert.True(t, after.verified,
				"clicking the link must record the address as verified even when MFA interrupts session issuance — otherwise passkey login and every other email_verified gate rejects the user permanently")
		})
	}
}

// TestMagicLinkSignupVerifiesEmail covers the passwordless entry point: a magic
// link creates the account and the click is the only proof of control there
// will ever be, so it must verify the address.
func TestMagicLinkSignupVerifiesEmail(t *testing.T) {
	for _, mfaOn := range []bool{false, true} {
		t.Run(fmt.Sprintf("mfa=%v", mfaOn), func(t *testing.T) {
			cfg := getTestConfig()
			cfg.IsEmailServiceEnabled = true
			cfg.EnableEmailVerification = true
			cfg.EnableMagicLinkLogin = true
			cfg.DisableMFA = !mfaOn
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "matrix_magic_" + uuid.NewString() + "@authorizer.dev"
			_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{Email: email})
			require.NoError(t, err)

			vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
			require.NoError(t, err)

			_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: vr.Token})
			require.NoError(t, err)

			state := readVerificationState(t, ts, ctx, email)
			require.True(t, state.exists)
			assert.True(t, state.verified,
				"a magic-link click is the only proof of control this account will ever produce; if it does not verify, the account is stuck")
			assert.Contains(t, state.methods, constants.AuthRecipeMethodMagicLinkLogin)
		})
	}
}

// TestVerifiedAccountCanUsePasskey closes the loop on the reported bug: it
// asserts the actual downstream consequence, not just the column.
func TestVerifiedAccountCanUsePasskey(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "matrix_passkey_" + uuid.NewString() + "@authorizer.dev"
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

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, user.EmailVerifiedAt,
		"webauthn.go refuses passkey login outright when this is nil, with an error the user cannot act on")
}

// TestLoginBeforeAndAfterVerification pins that an unverified account cannot
// simply log in, and that verifying unblocks it — the two halves have to agree
// or users get stuck on one side or let through on the other.
func TestLoginBeforeAndAfterVerification(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.DisableMFA = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "matrix_login_" + uuid.NewString() + "@authorizer.dev"
	const password = "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	// Unverified: the password is correct, but the account is not usable yet.
	// With the email service on this diverts into an OTP challenge rather than
	// a flat denial — either way it must NOT be a completed session.
	res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	if err == nil {
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "an unverified account must not receive a session from a plain password login")
	}

	vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)
	_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: vr.Token})
	require.NoError(t, err)

	// Verified: the same credentials now complete.
	res, err = ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotNil(t, res.AccessToken, "a verified account must be able to log in with the same password")
}

// TestResendAcrossIdentifiers exercises the recovery endpoint for every
// identifier it legitimately serves, in both the pending and no-pending states.
func TestResendAcrossIdentifiers(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.EnableMagicLinkLogin = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("signup identifier, request still pending", func(t *testing.T) {
		email := "matrix_resend_pending_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: "Password@123", ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)
		original, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.ResendVerifyEmail(ctx, &model.ResendVerifyEmailRequest{
			Email: email, Identifier: constants.VerificationTypeBasicAuthSignup,
		})
		require.NoError(t, err)

		fresh, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		assert.NotEqual(t, original.Token, fresh.Token, "a resend must rotate the token, not re-send the old one")
	})

	t.Run("signup identifier, no request pending", func(t *testing.T) {
		// The state a post-expiry password login leaves behind, since that path
		// purges the stale row. This used to silently do nothing, leaving the
		// user with no way to verify at all.
		email := "matrix_resend_gone_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: "Password@123", ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)
		vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		require.NoError(t, ts.StorageProvider.DeleteVerificationRequest(ctx, vr))

		_, err = ts.GraphQLProvider.ResendVerifyEmail(ctx, &model.ResendVerifyEmailRequest{
			Email: email, Identifier: constants.VerificationTypeBasicAuthSignup,
		})
		require.NoError(t, err)

		fresh, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err, "a fresh request must be minted when none is pending")
		require.NotEmpty(t, fresh.Token)

		// And it completes the flow end to end.
		_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: fresh.Token})
		require.NoError(t, err)
		assert.True(t, readVerificationState(t, ts, ctx, email).verified)
	})

	t.Run("already verified is a no-op", func(t *testing.T) {
		// The open-mailer guard: minting on demand must not let anyone who
		// knows a registered address make us send them mail.
		email := "matrix_resend_verified_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: "Password@123", ConfirmPassword: "Password@123",
		})
		require.NoError(t, err)
		vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: vr.Token})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.ResendVerifyEmail(ctx, &model.ResendVerifyEmailRequest{
			Email: email, Identifier: constants.VerificationTypeBasicAuthSignup,
		})
		require.NoError(t, err, "the response stays generic so it is not an existence oracle")

		_, err = ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.Error(t, err, "no request may be minted for an address that is already verified")
	})
}

// TestMobileSignupVerificationIsIndependent pins that the phone leg has its own
// verified flag and does not accidentally satisfy the email one — they gate
// different things (passkey login reads email, mobile OTP login reads phone).
func TestMobileSignupVerificationIsIndependent(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	cfg.EnablePhoneVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	mobile := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		PhoneNumber:     &mobile,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByPhoneNumber(ctx, mobile)
	require.NoError(t, err)
	assert.Nil(t, user.PhoneNumberVerifiedAt, "the phone is unverified until the OTP is redeemed")
	assert.Contains(t, user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth)
	// No email was ever supplied, so nothing should have marked one verified.
	assert.Empty(t, refs.StringValue(user.Email))
}

// TestVerifyEmailRESTMarksVerifiedBeforeMFAGate is the REST twin of
// TestVerifyEmailMarksVerifiedBeforeMFAGate, and the one that reproduces the
// bug a user actually hit.
//
// GET /verify_email is what the button in the verification email literally
// points to — a browser click never touches the GraphQL mutation. The two are
// SEPARATE implementations (internal/http_handlers/verify_email.go vs
// internal/service/verify_email.go), so fixing the service left the path every
// real user takes still broken.
//
// The write sat after the MFA gate's withheld branch, which redirects to MFA
// setup and returns. With MFA on by default, a fresh signup clicking its link
// lands there and the address is never marked verified.
//
// It stayed hidden because only passkey login checks the column: password,
// TOTP and email/SMS-OTP logins all succeed against an unverified account, so
// the flow looks fine until someone enrolls a passkey and is told - with no way
// to act on it - that their email is not verified.
func TestVerifyEmailRESTMarksVerifiedBeforeMFAGate(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	// Left at the derived default (true). This is the configuration that
	// triggers the bug — the MFA gate has to withhold for the early return to
	// be reached at all.
	cfg.EnableMFA = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_rest_" + uuid.NewString() + "@authorizer.dev"
	// The test config's AllowedOrigins is an explicit allowlist, and the handler
	// re-validates the token's redirect_uri against it on the click. Anything
	// else 400s on the redirect check before reaching the code under test.
	redirectURI := "http://localhost:3000"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
		RedirectURI:     &redirectURI,
	})
	require.NoError(t, err)

	before, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.Nil(t, before.EmailVerifiedAt)

	vr, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	// Click the link as the browser does, but stop at the first response
	// instead of chasing the redirect to the app origin (nothing is listening
	// there in a test).
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(testAuthorizerHost(ts) + "/verify_email?token=" + url.QueryEscape(vr.Token))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Confirm we actually exercised the branch that used to skip the write:
	// the MFA gate withheld the token and redirected to setup.
	require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Location"), "mfa_gate=offer",
		"this test is only meaningful if the MFA gate withheld — that is the early return the write used to sit behind")

	after, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, after.EmailVerifiedAt,
		"clicking the emailed link must record the address as verified even when the MFA gate withholds the token — otherwise passkey login rejects the user permanently, while password/TOTP/OTP logins hide it by never checking")
	assert.Equal(t, before.ID, after.ID)
}

// TestTOTPEnrollmentWorksForPhoneOnlyAccounts reproduces a mobile-signup bug:
// enrolling TOTP failed with pquerna/otp's "AccountName must be set".
//
// AccountName is the label the authenticator app shows, and the enrolment used
// the user's email verbatim. A phone-only signup has no email, so the value was
// empty and the library refused. MFA is on by default, so this is the first
// thing a mobile signup hits after verifying — the account could be created but
// never finish MFA setup.
//
// Asserted across all three identifier shapes because the fallback chain is
// email -> phone -> id, and only the middle one is exercised by mobile signup.
func TestTOTPEnrollmentWorksForPhoneOnlyAccounts(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be available for this test")
	_, ctx := createContext(ts)

	for _, tc := range []struct {
		name  string
		build func() *schemas.User
	}{
		{
			name: "phone only — the reported case",
			build: func() *schemas.User {
				phone := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
				return &schemas.User{
					PhoneNumber:   &phone,
					SignupMethods: constants.AuthRecipeMethodMobileBasicAuth,
				}
			},
		},
		{
			name: "email only",
			build: func() *schemas.User {
				email := "totp_email_" + uuid.NewString() + "@authorizer.dev"
				return &schemas.User{
					Email:         &email,
					SignupMethods: constants.AuthRecipeMethodBasicAuth,
				}
			},
		},
		{
			name: "neither — falls back to the user id",
			build: func() *schemas.User {
				// Not a shape signup produces today, but the fallback must not
				// depend on that staying true.
				return &schemas.User{SignupMethods: constants.AuthRecipeMethodBasicAuth}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user, err := ts.StorageProvider.AddUser(ctx, tc.build())
			require.NoError(t, err)

			authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
			require.NoError(t, err,
				"TOTP enrolment must not fail on a missing identifier — pquerna/otp rejects an empty AccountName")
			require.NotNil(t, authConfig)
			assert.NotEmpty(t, authConfig.Secret, "a usable secret must be issued")
			assert.NotEmpty(t, authConfig.ScannerImage, "the QR code is what the user actually scans")
		})
	}
}

// TestExpiredVerificationLinkIsRefusedAndRecoverable covers the case a real user
// hits most often: they did not click the link within 30 minutes.
//
// Two things must both hold, and only the pair is useful. The stale link must be
// refused — it is a capability, and an expired one that still works is just a
// long-lived one. And the user must be able to get a fresh link WITHOUT an
// admin, or "your link expired" is a dead end.
//
// The 30 minutes comes from CreateVerificationToken's `exp` claim
// (internal/token/verification_token.go); expiry is enforced by JWT validation
// inside ConsumeEmailVerificationToken, not by the row's expires_at column.
func TestExpiredVerificationLinkIsRefusedAndRecoverable(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.DisableMFA = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "expired_link_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)

	original, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	// Mint a token that is already past its exp, and swap it onto the stored
	// row so the row and the token agree — otherwise the lookup fails first and
	// the test would pass without ever exercising expiry.
	expiredToken, err := ts.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
		User:        &schemas.User{Email: &email},
		Nonce:       original.Nonce,
		HostName:    testAuthorizerHost(ts),
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	}, original.RedirectURI, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	expiredRow := *original
	expiredRow.Token = expiredToken
	expiredRow.ExpiresAt = time.Now().Add(-1 * time.Hour).Unix()
	require.NoError(t, ts.StorageProvider.DeleteVerificationRequest(ctx, original))
	_, err = ts.StorageProvider.AddVerificationRequest(ctx, &expiredRow)
	require.NoError(t, err)

	t.Run("an expired link is refused", func(t *testing.T) {
		// A genuinely expired token, signed with exp in the past. Built from
		// claims directly rather than CreateVerificationToken, which hardcodes
		// exp to +30m — there is no way to test expiry through that helper
		// without actually waiting, and a test that waits 30 minutes is a test
		// nobody runs.
		expiredClaims := jwt.MapClaims{
			"iss":          testAuthorizerHost(ts),
			"aud":          ts.Config.ClientID,
			"sub":          email,
			"exp":          time.Now().Add(-1 * time.Minute).Unix(),
			"iat":          time.Now().Add(-31 * time.Minute).Unix(),
			"token_type":   constants.VerificationTypeBasicAuthSignup,
			"nonce":        expiredRow.Nonce,
			"redirect_uri": expiredRow.RedirectURI,
		}
		signed, sErr := ts.TokenProvider.SignJWTToken(expiredClaims)
		require.NoError(t, sErr)

		// Put it on the stored row so the lookup succeeds and the request
		// genuinely reaches the expiry check, rather than failing earlier as an
		// unknown token — which would pass for the wrong reason.
		row := expiredRow
		row.Token = signed
		require.NoError(t, ts.StorageProvider.DeleteVerificationRequest(ctx, &expiredRow))
		_, aErr := ts.StorageProvider.AddVerificationRequest(ctx, &row)
		require.NoError(t, aErr)
		expiredRow = row

		_, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: signed})
		require.Error(t, err, "an expired verification link must not be redeemable — it is a capability, and an expired one that still works is just a long-lived one")
		assert.Contains(t, err.Error(), "invalid verification token")

		user, uErr := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, uErr)
		assert.Nil(t, user.EmailVerifiedAt, "a refused link must not verify the address")
	})

	t.Run("a stale link whose nonce no longer matches is refused", func(t *testing.T) {
		// This is what an expired-then-resent link looks like in practice: the
		// resend rotates the nonce, so the OLD link stops validating even before
		// its exp. Same refusal path, and testable without waiting 30 minutes.
		_, err := ts.GraphQLProvider.ResendVerifyEmail(ctx, &model.ResendVerifyEmailRequest{
			Email:      email,
			Identifier: constants.VerificationTypeBasicAuthSignup,
		})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: expiredRow.Token})
		require.Error(t, err, "the superseded link must stop working once a new one is issued")
		assert.Contains(t, err.Error(), "invalid verification token")

		user, uErr := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, uErr)
		assert.Nil(t, user.EmailVerifiedAt, "a refused link must not verify the address")
	})

	t.Run("the resent link works, so the user is never stuck", func(t *testing.T) {
		fresh, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		require.NotEqual(t, expiredRow.Token, fresh.Token)

		_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: fresh.Token})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, user.EmailVerifiedAt, "the recovery path must actually complete verification")
	})
}

// TestVerificationTokenWithEmptySubjectIsRefused pins a guard that does not
// depend on anything else being right.
//
// ValidateJWTClaims checks `sub` as
//
//	claims["sub"] != cfg.User.ID && claims["sub"] != cfg.User.Email
//
// and the verification path sets only Email, leaving User.ID as "". A token
// whose `sub` is the empty STRING therefore satisfies the first comparison and
// passes that check. Whether it then does damage depends on whether the storage
// backend keeps a missing email as NULL or as "" — SQL keeps it NULL, so the
// lookup finds nobody, but that is an accident of one backend's representation
// across six implementations, not a declared property.
//
// So the subject is rejected outright. If this ever fails, an empty-subject
// token is reaching account selection and the only thing standing between it
// and a real account is a per-backend NULL convention.
func TestVerificationTokenWithEmptySubjectIsRefused(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "empty_sub_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        "Password@123",
		ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)

	row, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	// Everything valid except the subject, which is empty.
	forged, err := ts.TokenProvider.SignJWTToken(jwt.MapClaims{
		"iss":          testAuthorizerHost(ts),
		"aud":          ts.Config.ClientID,
		"sub":          "",
		"exp":          time.Now().Add(10 * time.Minute).Unix(),
		"iat":          time.Now().Unix(),
		"token_type":   constants.VerificationTypeBasicAuthSignup,
		"nonce":        row.Nonce,
		"redirect_uri": row.RedirectURI,
	})
	require.NoError(t, err)

	// Attach it to the stored row so the lookup succeeds and the request really
	// reaches subject handling instead of failing earlier as an unknown token.
	forgedRow := *row
	forgedRow.Token = forged
	require.NoError(t, ts.StorageProvider.DeleteVerificationRequest(ctx, row))
	_, err = ts.StorageProvider.AddVerificationRequest(ctx, &forgedRow)
	require.NoError(t, err)

	_, err = ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{Token: forged})
	require.Error(t, err, "an empty subject must never be used to select an account")
	assert.Contains(t, err.Error(), "invalid verification token")

	// And nothing was verified as a side effect.
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	assert.Nil(t, user.EmailVerifiedAt)
}
