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

// TestLockMFA covers: locking with a valid mfa session and no OTP fallback
// succeeds and blocks subsequent login; a caller with no valid mfa session
// is rejected with Unauthenticated.
func TestLockMFA(t *testing.T) {
	const password = "Password@123"

	t.Run("locks the account and blocks subsequent login", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "lock_mfa_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// LockMFA is reached mid-MFA-flow, identified by the mfa session
		// cookie plus email — same identification pattern as SkipMFASetup.
		// Set the session directly rather than driving a full TOTP/OTP
		// challenge; LockMFA itself never issues a token, so there is
		// nothing about a real challenge this test needs to exercise.
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		lockRes, err := ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, lockRes)

		locked, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, locked.MFALockedAt, "lock_mfa must persist MFALockedAt")

		// The signup password is real (unlike a bare AddUser fixture), so a
		// rejection here is unambiguously the lockout check, not an
		// incidental bad-password mismatch.
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.Error(t, err)
		assert.Nil(t, loginRes)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindFailedPrecondition, svcErr.Kind, "a locked account must be rejected by the lockout check specifically, not any other error kind")
	})

	t.Run("refuses to lock when a verified SMS-OTP fallback exists", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "lock_mfa_otp_fallback_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// A verified SMS-OTP authenticator is a working recovery path, so
		// locking must be refused: the user should use it instead.
		_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     user.ID,
			Method:     constants.EnvKeySMSOTPAuthenticator,
			VerifiedAt: &now,
		})
		require.NoError(t, err)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		lockRes, err := ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{Email: &email})
		require.Error(t, err)
		assert.Nil(t, lockRes)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindFailedPrecondition, svcErr.Kind, "a verified OTP fallback must block locking with FailedPrecondition")

		unlocked, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Nil(t, unlocked.MFALockedAt, "a refused lock must not have persisted MFALockedAt")
	})

	for _, enforceMFA := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejects with Unauthenticated when caller has no valid mfa session (EnforceMFA=%v)", enforceMFA), func(t *testing.T) {
			cfg := getTestConfig()
			cfg.EnableMFA = true
			cfg.EnforceMFA = enforceMFA
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "lock_mfa_nosession_" + uuid.NewString() + "@authorizer.dev"
			now := time.Now().Unix()
			_, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
				Email:                    refs.NewStringRef(email),
				EmailVerifiedAt:          &now,
				SignupMethods:            constants.AuthRecipeMethodBasicAuth,
				IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
			})
			require.NoError(t, err)

			lockRes, err := ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{Email: &email})
			require.Error(t, err)
			assert.Nil(t, lockRes)

			var svcErr *service.Error
			require.True(t, errors.As(err, &svcErr))
			assert.Equal(t, service.KindUnauthenticated, svcErr.Kind)
		})
	}

	t.Run("rejects a Challenge session (ResendOTP/ForgotPassword) with Unauthenticated", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "lock_mfa_challenge_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// The pre-auth account-lockout DoS: an attacker who only knows the
		// victim's email obtains a Challenge session via ResendOTP, then tries
		// to permanently lock the account. It must be rejected.
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		lockRes, err := ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{Email: &email})
		require.Error(t, err)
		assert.Nil(t, lockRes)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindUnauthenticated, svcErr.Kind, "a Challenge session must not be able to lock an account")

		unlocked, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Nil(t, unlocked.MFALockedAt, "a rejected Challenge session must not have locked the account")
	})
}
