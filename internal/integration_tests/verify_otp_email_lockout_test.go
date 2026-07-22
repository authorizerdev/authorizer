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

// TestVerifyOTPEmailLockout guards the per-user email/SMS OTP verification
// lockout — the mirror of the TOTP lockout (see verify_otp_totp_lockout_test.go).
// Five failed attempts within the window lock verification, after which even the
// correct code is refused with the distinct lockout error (not the generic
// "invalid otp"), and a successful verification resets the counter.
//
// This defends the account against a brute-force that spreads guesses across
// many IPs to slip under the global per-IP rate limiter — the same threat the
// TOTP branch already mitigated, previously left open on the email/SMS path.
func TestVerifyOTPEmailLockout(t *testing.T) {
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

	const otpLockoutCachePrefix = "otp_failed_attempts:"
	const maxFailedAttempts = 5
	const knownPlainOTP = "123456"

	mkUser := func(t *testing.T) (*schemas.User, string) {
		t.Helper()
		email := "verify_otp_lockout_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:           refs.NewStringRef(email),
			EmailVerifiedAt: &now,
			SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		})
		require.NoError(t, err)
		return user, email
	}

	// seedOTP writes a known plaintext/digest OTP row (the suite cannot
	// intercept the outgoing email, so we plant a code we know).
	seedOTP := func(t *testing.T, email string) {
		t.Helper()
		_, err := ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:     email,
			Otp:       crypto.HashOTP(knownPlainOTP, cfg.JWTSecret),
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		})
		require.NoError(t, err)
	}

	armMfaSession := func(userID string) {
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(userID, mfaSession,
			constants.MFASessionPurposeVerified,
			time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
	}

	verify := func(email, otp string) error {
		_, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email: &email,
			Otp:   otp,
		})
		return err
	}

	t.Run("five failures lock verification; a correct code is then refused with a distinct error", func(t *testing.T) {
		user, email := mkUser(t)
		seedOTP(t, email)

		// Five failed attempts with a wrong code — each the generic
		// invalid-otp rejection, not the lockout error.
		for i := 0; i < maxFailedAttempts; i++ {
			armMfaSession(user.ID)
			err := verify(email, "000000")
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "too many failed attempts",
				"attempt %d must still be a normal rejection, not a lockout", i+1)
		}

		// Sixth attempt with the CORRECT code must be refused: the account is
		// now locked, so the distinct lockout error is returned, not success.
		armMfaSession(user.ID)
		err := verify(email, knownPlainOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many failed attempts",
			"a correct code offered while locked must return the lockout error")
	})

	t.Run("successful verification resets the failed-attempt counter", func(t *testing.T) {
		user, email := mkUser(t)
		seedOTP(t, email)
		lockKey := otpLockoutCachePrefix + user.ID

		// Two failed attempts prime the counter.
		for i := 0; i < 2; i++ {
			armMfaSession(user.ID)
			require.Error(t, verify(email, "000000"))
		}
		counter, err := ts.MemoryStoreProvider.GetCache(lockKey)
		require.NoError(t, err)
		assert.Equal(t, "2", counter, "two failures must be recorded")

		// A successful verification must clear the counter.
		armMfaSession(user.ID)
		require.NoError(t, verify(email, knownPlainOTP))

		counter, err = ts.MemoryStoreProvider.GetCache(lockKey)
		require.NoError(t, err)
		assert.Empty(t, counter, "a successful verification must reset the failed-attempt counter")
	})
}
