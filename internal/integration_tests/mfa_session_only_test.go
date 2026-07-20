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
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestMFASessionOnlyContinuation covers the OAuth-return MFA continuation path:
// the caller has a valid MFA session cookie but supplies NO email/phone_number
// (the identifier never travels in the OAuth redirect). Each continuation
// endpoint must then resolve the account from the session alone via
// MemoryStoreProvider.GetMfaSessionOwner and behave exactly as the
// identifier-supplied path would — while still rejecting a bare Challenge
// session, preserving the existing account-lockout-DoS guarantee.
func TestMFASessionOnlyContinuation(t *testing.T) {
	t.Run("SkipMFASetup resolves the account from the session cookie alone", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "session_only_skip_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// Mirror what EvaluateMFAGateForOAuth leaves behind: a Verified session
		// keyed by the user, and the cookie on the request — but NO identifier
		// in the request params.
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{})
		require.NoError(t, err)
		require.NotNil(t, skipRes)
		require.NotNil(t, skipRes.AccessToken, "session-only skip must issue the withheld token, same as the identifier path")
		assert.NotEmpty(t, *skipRes.AccessToken)

		updated, err := ts.StorageProvider.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, updated.HasSkippedMFASetupAt, "skip must persist HasSkippedMFASetupAt on the SAME user resolved from the session")
	})

	t.Run("VerifyOTP resolves the account from the session cookie alone", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "session_only_verify_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// Seed an email OTP keyed by the user's email — the session-only path
		// derives the email from the resolved account, so this is what the
		// verify branch will look up.
		const plainOTP = "246810"
		_, err = ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:     email,
			Otp:       crypto.HashOTP(plainOTP, cfg.JWTSecret),
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		})
		require.NoError(t, err)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Otp: plainOTP})
		require.NoError(t, err)
		require.NotNil(t, verifyRes)
		require.NotNil(t, verifyRes.AccessToken, "session-only verify must issue a token")
		assert.NotEmpty(t, *verifyRes.AccessToken)
		if verifyRes.User != nil {
			assert.Equal(t, email, refs.StringValue(verifyRes.User.Email), "must resolve and authenticate the SAME user the session belongs to")
		}
	})

	t.Run("SkipMFASetup rejects a Challenge session with no identifier (Unauthenticated)", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "session_only_skip_chal_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// A Challenge session must not gain skip powers through the session-only
		// fallback either — same guarantee as the identifier path.
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{})
		require.Error(t, err)
		assert.Nil(t, skipRes)
		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindUnauthenticated, svcErr.Kind)

		unchanged, err := ts.StorageProvider.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Nil(t, unchanged.HasSkippedMFASetupAt, "a rejected Challenge session must not have recorded a skip")
	})

	t.Run("LockMFA rejects a Challenge session with no identifier (Unauthenticated)", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "session_only_lock_chal_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		lockRes, err := ts.GraphQLProvider.LockMFA(ctx, &model.LockMfaRequest{})
		require.Error(t, err)
		assert.Nil(t, lockRes)
		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindUnauthenticated, svcErr.Kind)

		unchanged, err := ts.StorageProvider.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Nil(t, unchanged.MFALockedAt, "a rejected Challenge session must not have locked the account")
	})

	t.Run("ResendOTP with a Verified session and no identifier resends for the right user", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableEmailOTP = true
		cfg.IsEmailServiceEnabled = true
		cfg.EnableEmailVerification = true
		cfg.SMTPHost = "localhost"
		cfg.SMTPPort = 1025
		cfg.SMTPSenderEmail = "test@authorizer.dev"
		cfg.SMTPSenderName = "Test"
		cfg.SMTPLocalName = "Test"
		cfg.SMTPSkipTLSVerification = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "session_only_resend_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// ResendOTP requires an existing OTP row to resend (unchanged existing
		// behavior); seed one keyed by the resolved account's email.
		_, err = ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:     email,
			Otp:       crypto.HashOTP("111111", cfg.JWTSecret),
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		})
		require.NoError(t, err)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		resendRes, err := ts.GraphQLProvider.ResendOTP(ctx, &model.ResendOTPRequest{})
		require.NoError(t, err)
		require.NotNil(t, resendRes)
		assert.Equal(t, "OTP has been sent. Please check your inbox", resendRes.Message)
	})

	t.Run("ResendOTP with a Challenge session and no identifier is rejected as InvalidArgument", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableEmailOTP = true
		cfg.IsEmailServiceEnabled = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "session_only_resend_chal_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// Only a Verified session unlocks the session-only resend. A Challenge
		// session with no identifier is treated exactly as if no identifier was
		// supplied — InvalidArgument, so it cannot spawn further resends.
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		resendRes, err := ts.GraphQLProvider.ResendOTP(ctx, &model.ResendOTPRequest{})
		require.Error(t, err)
		assert.Nil(t, resendRes)
		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindInvalidArgument, svcErr.Kind)
	})
}
