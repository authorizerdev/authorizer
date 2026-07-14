package integration_tests

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestSkipMFASetup covers the security-relevant behaviors of the
// skip_mfa_setup mutation under the withheld-token model:
//   - a valid MFA session + matching email, with MFA optional, records
//     HasSkippedMFASetupAt and issues the previously-withheld access token.
//   - with EnforceMFA=true it is rejected with FailedPrecondition even with
//     a valid MFA session — enforcement is never skippable.
//   - with no valid MFA session cookie at all, it is rejected with
//     Unauthenticated in both EnforceMFA states.
func TestSkipMFASetup(t *testing.T) {
	const password = "Password@123"

	t.Run("skips setup, issues the withheld token, and quiets a later login", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "skip_mfa_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.Nil(t, loginRes.AccessToken, "first login with optional MFA and no prior enrollment/skip must withhold the token")
		require.True(t, refs.BoolValue(loginRes.ShouldShowTotpScreen))

		// Login withholds the token behind an MFA session cookie set on the
		// response (Set-Cookie), not on the request — http.Request cookies
		// are not auto-updated from responses in this in-process test setup
		// (see latestAppSessionCookie's doc comment). Every other MFA-session
		// test in this package (verify_otp_totp_test.go,
		// verify_otp_totp_lockout_test.go, webauthn_test.go) copies the
		// cookie onto the request by hand for the same reason; mirror that
		// here rather than relying on it propagating automatically.
		mfaSession := latestMfaSessionCookie(ts)
		require.NotEmpty(t, mfaSession, "login must have set an mfa session cookie on the response")
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, skipRes)
		require.NotNil(t, skipRes.AccessToken, "skip must issue the token that was withheld at login")
		assert.NotEmpty(t, *skipRes.AccessToken)

		updated, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, updated.HasSkippedMFASetupAt, "skip_mfa_setup must persist HasSkippedMFASetupAt")

		secondLogin, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, secondLogin.AccessToken, "a user who already skipped setup must log in normally, token issued immediately")
	})

	t.Run("rejects with FailedPrecondition when MFA is enforced, even with a valid mfa session", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "skip_mfa_enforced_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.Error(t, err)
		assert.Nil(t, skipRes)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindFailedPrecondition, svcErr.Kind, "EnforceMFA must reject with FailedPrecondition, not Unauthenticated or any other kind")
	})

	for _, enforceMFA := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejects with Unauthenticated when caller has no valid mfa session (EnforceMFA=%v)", enforceMFA), func(t *testing.T) {
			cfg := getTestConfig()
			cfg.EnableMFA = true
			cfg.EnableTOTPLogin = true
			cfg.EnforceMFA = enforceMFA
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "skip_mfa_nosession_" + uuid.NewString() + "@authorizer.dev"
			now := time.Now().Unix()
			_, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
				Email:                    refs.NewStringRef(email),
				EmailVerifiedAt:          &now,
				SignupMethods:            constants.AuthRecipeMethodBasicAuth,
				IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
			})
			require.NoError(t, err)

			skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
			require.Error(t, err)
			assert.Nil(t, skipRes)

			var svcErr *service.Error
			require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
			assert.Equal(t, service.KindUnauthenticated, svcErr.Kind, "a caller with no valid mfa session must get Unauthenticated regardless of EnforceMFA")
		})
	}

	t.Run("rejects with InvalidArgument when neither email nor phone_number is given", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		// SkipMFASetup reads the mfa session cookie before validating
		// email/phone_number (mirrors VerifyOTP's ordering), so a cookie
		// must be present here or the call short-circuits on Unauthenticated
		// before ever reaching the check this subtest targets. The value
		// itself need not resolve to a real session — no user lookup happens
		// before the email/phone_number check runs.
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", uuid.NewString()))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{})
		require.Error(t, err)
		assert.Nil(t, skipRes)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr))
		assert.Equal(t, service.KindInvalidArgument, svcErr.Kind)
	})
}
