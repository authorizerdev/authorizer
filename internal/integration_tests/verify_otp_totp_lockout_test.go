package integration_tests

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestVerifyOTPTOTPLockout guards the per-user TOTP verification lockout: five
// failed attempts within the window lock verification, after which even a
// correct code is refused with a distinct lockout error (not the generic
// "invalid otp"), and a successful verification resets the counter.
//
// This defends the account against a brute-force that spreads guesses across
// many IPs to slip under the global per-IP rate limiter.
func TestVerifyOTPTOTPLockout(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
	req, ctx := createContext(ts)

	const lockoutCachePrefix = "totp_failed_attempts:"
	const maxFailedAttempts = 5

	mkVerifiedTOTPUser := func(t *testing.T) (*schemas.User, string, string) {
		t.Helper()
		email := "verify_totp_lockout_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:           refs.NewStringRef(email),
			EmailVerifiedAt: &now,
			SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		})
		require.NoError(t, err)
		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)
		return user, email, authConfig.Secret
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
			Email:  &email,
			Otp:    otp,
			IsTotp: refs.NewBoolRef(true),
		})
		return err
	}

	t.Run("five failures lock verification; a correct code is then refused with a distinct error", func(t *testing.T) {
		user, email, secret := mkVerifiedTOTPUser(t)

		// Five failed attempts with a wrong code. Each is the generic
		// invalid-otp error, not the lockout error.
		for i := 0; i < maxFailedAttempts; i++ {
			armMfaSession(user.ID)
			err := verify(email, "000000")
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "too many failed attempts",
				"attempt %d must still be a normal rejection, not a lockout", i+1)
		}

		// Sixth attempt with a CORRECT code must be refused because the
		// account is now locked — the distinct lockout error, not success.
		armMfaSession(user.ID)
		validCode, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		err = verify(email, validCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many failed attempts",
			"a correct code offered while locked must return the lockout error")
	})

	t.Run("successful verification resets the failed-attempt counter", func(t *testing.T) {
		user, email, secret := mkVerifiedTOTPUser(t)
		lockKey := lockoutCachePrefix + user.ID

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
		validCode, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		require.NoError(t, verify(email, validCode))

		counter, err = ts.MemoryStoreProvider.GetCache(lockKey)
		require.NoError(t, err)
		assert.Empty(t, counter, "a successful verification must reset the failed-attempt counter")
	})

	t.Run("concurrent failed attempts cannot bypass the lockout via a check-then-increment race", func(t *testing.T) {
		// Guards against a non-atomic read-modify-write counter: if the
		// implementation reads the count, compares it, THEN writes count+1,
		// concurrent requests can all read the same pre-increment value and
		// all pass the check, so the counter only ever advances by 1 per
		// concurrent burst regardless of how many requests race - the lockout
		// never engages no matter how many guesses are fired. With an atomic
		// increment-then-check, IncrementCache must instead hand out strictly
		// increasing, unique values, so at most maxFailedAttempts requests
		// can ever reach validation and the counter must land exactly at the
		// number of concurrent attempts - not less.
		user, email, _ := mkVerifiedTOTPUser(t)
		lockKey := lockoutCachePrefix + user.ID

		// One MFA session shared by every concurrent request, matching a real
		// client re-submitting several attempts within one login flow. Set
		// once before firing concurrently: only reads happen after this, so
		// there's no header-mutation race in the test itself.
		armMfaSession(user.ID)

		const concurrency = 20
		var wg sync.WaitGroup
		var lockedCount, rejectedCount int64
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := verify(email, "000000")
				// assert (not require) inside a goroutine: require's FailNow
				// is only safe to call from the test's own goroutine.
				if !assert.Error(t, err) {
					return
				}
				if strings.Contains(err.Error(), "too many failed attempts") {
					atomic.AddInt64(&lockedCount, 1)
				} else {
					atomic.AddInt64(&rejectedCount, 1)
				}
			}()
		}
		wg.Wait()

		// Every one of the `concurrency` requests must have actually
		// incremented the counter - a check-then-increment race would leave
		// this well short of `concurrency`.
		counter, err := ts.MemoryStoreProvider.GetCache(lockKey)
		require.NoError(t, err)
		counterVal, convErr := strconv.Atoi(counter)
		require.NoError(t, convErr)
		assert.Equal(t, concurrency, counterVal,
			"the counter must reflect every concurrent attempt - a lost update here is exactly the race this test guards against")

		// At most maxFailedAttempts requests may ever reach real validation
		// (and so get the generic invalid-otp rejection); everything beyond
		// that must be turned away by the lockout before validating.
		assert.LessOrEqual(t, int(rejectedCount), maxFailedAttempts,
			"no more than maxFailedAttempts requests may bypass the lockout and reach validation")
		assert.Equal(t, int64(concurrency)-rejectedCount, lockedCount)
	})
}
