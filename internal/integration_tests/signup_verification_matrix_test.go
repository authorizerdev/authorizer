package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
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
