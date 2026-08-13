package integration_tests

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestTOTPRecoveryCodesAtRest covers the at-rest hardening for TOTP recovery
// codes:
//
//   - Generate() returns the plaintext recovery codes to the caller once (the
//     frontend shows them to the user), but persists only their SHA-256
//     hashes — a DB dump must never reveal a usable recovery code.
//   - ValidateRecoveryCode() hashes the incoming code, matches it against the
//     stored hashes, and consumes it exactly once.
//   - Legacy plaintext recovery-code rows written by a pre-hashing release
//     still validate once via the plaintext fallback path.
func TestTOTPRecoveryCodesAtRest(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableTOTPLogin = true
	cfg.EnableMFA = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
	ctx := context.Background()

	mkUser := func(t *testing.T) *schemas.User {
		t.Helper()
		email := "totp_recovery_" + uuid.NewString() + "@authorizer.dev"
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email: refs.NewStringRef(email),
		})
		require.NoError(t, err)
		return user
	}

	t.Run("Generate persists only hashes, never plaintext recovery codes", func(t *testing.T) {
		user := mkUser(t)
		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)
		require.Len(t, authConfig.RecoveryCodes, 10)

		row, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, user.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		stored := refs.StringValue(row.RecoveryCodes)

		// No plaintext code may appear anywhere in the persisted blob.
		for _, code := range authConfig.RecoveryCodes {
			assert.NotContains(t, stored, code,
				"plaintext recovery code must not be persisted")
		}

		// Every stored map key must be a SHA-256 hash, and the hash of each
		// plaintext code must be present.
		storedMap := map[string]bool{}
		require.NoError(t, json.Unmarshal([]byte(stored), &storedMap))
		require.Len(t, storedMap, 10)
		for key := range storedMap {
			assert.True(t, crypto.IsHashedRecoveryCode(key),
				"stored recovery-code key %q must be a SHA-256 hash", key)
		}
		for _, code := range authConfig.RecoveryCodes {
			_, ok := storedMap[crypto.HashRecoveryCode(code)]
			assert.True(t, ok, "hash of plaintext code must be a stored key")
		}
	})

	t.Run("ValidateRecoveryCode accepts a hashed code exactly once", func(t *testing.T) {
		user := mkUser(t)
		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)

		code := authConfig.RecoveryCodes[0]
		ok, err := ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, code, user.ID)
		require.NoError(t, err)
		assert.True(t, ok, "a freshly generated recovery code must validate")

		// One-time use: the same code must not validate a second time.
		ok, err = ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, code, user.ID)
		require.NoError(t, err)
		assert.False(t, ok, "a consumed recovery code must not validate again")

		// A code that was never issued must be rejected (no error).
		ok, err = ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, uuid.NewString(), user.ID)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("legacy plaintext recovery code validates once via fallback", func(t *testing.T) {
		user := mkUser(t)

		// Simulate a row enrolled on a pre-hashing release: recovery codes
		// stored as plaintext UUID keys.
		legacyCode := uuid.NewString()
		legacyMap := map[string]bool{legacyCode: false}
		blob, err := json.Marshal(legacyMap)
		require.NoError(t, err)
		_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			Secret:        "enc:v1:placeholder", // secret is irrelevant to recovery-code validation
			RecoveryCodes: refs.NewStringRef(string(blob)),
			UserID:        user.ID,
			Method:        constants.EnvKeyTOTPAuthenticator,
		})
		require.NoError(t, err)

		// Sanity: the stored key really is legacy plaintext, not a hash.
		require.False(t, crypto.IsHashedRecoveryCode(legacyCode))
		require.False(t, strings.Contains(string(blob), crypto.HashRecoveryCode(legacyCode)))

		// The legacy plaintext code must validate once via the fallback path.
		ok, err := ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, legacyCode, user.ID)
		require.NoError(t, err)
		assert.True(t, ok, "legacy plaintext recovery code must validate during rolling upgrade")

		// And be consumed one-time, just like a hashed code.
		ok, err = ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, legacyCode, user.ID)
		require.NoError(t, err)
		assert.False(t, ok, "consumed legacy recovery code must not validate again")
	})

	// "Exactly once" has to hold under concurrency, not just in sequence. The
	// original implementation read the blob, decided in Go, and wrote it back
	// unconditionally, so racing redemptions of one code all saw it unconsumed
	// and all returned true — measured here at 3 of 8 before the fix. That
	// turns a single leaked recovery code into an unlimited supply of logins,
	// which is the one thing a recovery code must not be.
	t.Run("one recovery code redeemed concurrently succeeds exactly once", func(t *testing.T) {
		const racers = 8
		for round := 0; round < 25; round++ {
			user := mkUser(t)
			authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
			require.NoError(t, err)
			code := authConfig.RecoveryCodes[0]

			var start sync.WaitGroup
			var done sync.WaitGroup
			var accepted atomic.Int32
			start.Add(1)
			for i := 0; i < racers; i++ {
				done.Add(1)
				go func() {
					defer done.Done()
					start.Wait()
					ok, err := ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, code, user.ID)
					assert.NoError(t, err, "a lost race is a rejection, never a fault")
					if ok {
						accepted.Add(1)
					}
				}()
			}
			start.Done()
			done.Wait()

			require.Equal(t, int32(1), accepted.Load(),
				"round %d: exactly one concurrent redemption of a single recovery code may succeed", round)
		}
	})

	// Racing redemptions of DIFFERENT codes must both land. The blob is one
	// column, so a compare-and-swap that simply gave up would silently drop the
	// loser's code back to unconsumed — trading a double-spend for a lost
	// write. The retry re-reads and re-applies instead.
	t.Run("two different recovery codes redeemed concurrently both succeed", func(t *testing.T) {
		for round := 0; round < 25; round++ {
			user := mkUser(t)
			authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
			require.NoError(t, err)

			var start sync.WaitGroup
			var done sync.WaitGroup
			var accepted atomic.Int32
			start.Add(1)
			for i := 0; i < 2; i++ {
				code := authConfig.RecoveryCodes[i]
				done.Add(1)
				go func() {
					defer done.Done()
					start.Wait()
					ok, err := ts.AuthenticatorProvider.ValidateRecoveryCode(ctx, code, user.ID)
					assert.NoError(t, err)
					if ok {
						accepted.Add(1)
					}
				}()
			}
			start.Done()
			done.Wait()

			require.Equal(t, int32(2), accepted.Load(),
				"round %d: distinct recovery codes must not invalidate each other", round)

			// Both are spent, and the other eight are untouched.
			row, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
			require.NoError(t, err)
			storedMap := map[string]bool{}
			require.NoError(t, json.Unmarshal([]byte(refs.StringValue(row.RecoveryCodes)), &storedMap))
			require.Len(t, storedMap, 10, "no code may be lost by a concurrent write")
			spent := 0
			for _, consumed := range storedMap {
				if consumed {
					spent++
				}
			}
			assert.Equal(t, 2, spent, "round %d: exactly the two redeemed codes are marked consumed", round)

			// And neither of the remaining codes was collaterally burned.
			for _, code := range authConfig.RecoveryCodes[2:] {
				assert.False(t, storedMap[crypto.HashRecoveryCode(code)],
					"an unredeemed code must still be spendable")
			}
		}
	})
}
