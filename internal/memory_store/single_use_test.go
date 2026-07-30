package memory_store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage"
)

// TestGetAndRemoveStateIsSingleUse is the regression test for authorization-code
// replay. GetAndRemoveState is the primitive behind RFC 6749 §4.1.2 single-use
// codes (see http_handlers.TokenHandler) and the SSO broker's single-use
// `state`, so under concurrent redemption of the same key exactly one caller may
// receive the value.
//
// The DB-backed provider previously implemented this as a read followed by a
// separate delete, which handed the same code to every racer: 39 of 40 rounds
// returned it to BOTH callers. Runs against every provider so no future backend
// can regress the contract.
func TestGetAndRemoveStateIsSingleUse(t *testing.T) {
	for _, storeType := range memoryStoreTypesForTest() {
		t.Run(storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			if storeType == memoryStoreTypeRedis && err != nil {
				t.Skipf("skipping redis (is Redis running on localhost:6380?): %v", err)
			}
			require.NoError(t, err)

			const rounds = 40
			doubleRedemptions := 0

			for i := 0; i < rounds; i++ {
				key := "authcode_" + uuid.New().String()
				require.NoError(t, p.SetState(key, "session-data-for-"+key))

				var wg sync.WaitGroup
				results := make([]string, 2)
				errs := make([]error, 2)
				start := make(chan struct{})
				for j := range results {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						<-start
						results[idx], errs[idx] = p.GetAndRemoveState(key)
					}(j)
				}
				close(start)
				wg.Wait()

				won := 0
				for j := range results {
					if errs[j] == nil && results[j] != "" {
						won++
					}
				}
				assert.NotZero(t, won, "one caller must still succeed — the code must not be lost")
				if won > 1 {
					doubleRedemptions++
				}
			}

			assert.Zero(t, doubleRedemptions,
				"an authorization code must be redeemable exactly once (RFC 6749 §4.1.2); "+
					"%d of %d concurrent double-redemptions returned it to both callers", doubleRedemptions, rounds)
		})
	}
}

// TestClaimRefreshTokenIsSingleUse is the regression test for concurrent
// refresh-token redemption. ClaimRefreshToken is the gate the token endpoint
// must win before issuing a rotated token (OAuth 2.1 §6.1), so under concurrent
// redemption of the same nonce exactly one caller may observe true.
//
// Before this existed the endpoint validated (a READ) and then deleted, so two
// simultaneous refreshes both passed and each minted an independent, separately
// rotating token family from one refresh token — reproduced end to end as
// [200,200] by e2e-playground/tests/concurrency.spec.ts. Runs against every
// provider so no backend can regress the contract.
func TestClaimRefreshTokenIsSingleUse(t *testing.T) {
	for _, storeType := range memoryStoreTypesForTest() {
		t.Run(storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			if storeType == memoryStoreTypeRedis && err != nil {
				t.Skipf("skipping redis (is Redis running on localhost:6380?): %v", err)
			}
			require.NoError(t, err)

			const rounds = 40
			doubleClaims := 0
			expiry := time.Now().Add(10 * time.Minute).Unix()

			for i := 0; i < rounds; i++ {
				sessionKey := "basic_auth:user-" + uuid.New().String()
				nonce := uuid.New().String()
				require.NoError(t, p.SetUserSession(
					sessionKey, "refresh_token_"+nonce, "the-refresh-token", expiry))

				var wg sync.WaitGroup
				claims := make([]bool, 2)
				errs := make([]error, 2)
				start := make(chan struct{})
				for j := range claims {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						<-start
						claims[idx], errs[idx] = p.ClaimRefreshToken(sessionKey, nonce)
					}(j)
				}
				close(start)
				wg.Wait()

				won := 0
				for j := range claims {
					require.NoError(t, errs[j])
					if claims[j] {
						won++
					}
				}
				assert.NotZero(t, won, "one caller must still win — the token must not be lost")
				if won > 1 {
					doubleClaims++
				}
			}

			assert.Zero(t, doubleClaims,
				"a refresh token must be redeemable exactly once (OAuth 2.1 §6.1); "+
					"%d of %d concurrent double-redemptions were granted to both callers", doubleClaims, rounds)
		})
	}
}

// TestClaimRefreshTokenIsIdempotentlyFalse pins the absent-entry contract: a
// claim on a nonce that was never issued (or already consumed) is false with no
// error, never an error the endpoint would have to special-case.
func TestClaimRefreshTokenIsIdempotentlyFalse(t *testing.T) {
	for _, storeType := range memoryStoreTypesForTest() {
		t.Run(storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			if storeType == memoryStoreTypeRedis && err != nil {
				t.Skipf("skipping redis: %v", err)
			}
			require.NoError(t, err)

			claimed, err := p.ClaimRefreshToken("basic_auth:nobody", uuid.New().String())
			assert.NoError(t, err, "claiming an absent refresh token is not an error")
			assert.False(t, claimed)
		})
	}
}

// TestDBStoreCacheIsSharedAcrossInstances is the regression test for the
// process-local cache. The DB-backed provider is what New() selects for a
// multi-instance deployment that has a database but no REDIS_URL, so its cache
// must be visible to every replica: SAML assertion single-use, RFC 7523 jti
// replay defence, and OTP lockout counters are all built on these methods, and
// each silently degrades when a replica cannot see what another wrote.
//
// Two providers over ONE storage backend stand in for two replicas.
func TestDBStoreCacheIsSharedAcrossInstances(t *testing.T) {
	logger := zerolog.Nop()
	cfg := getTestMemoryStorageConfig(memoryStoreTypeDB)

	dbDir, err := os.MkdirTemp("", "authorizer-ms-shared-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dbDir) })
	cfg.DatabaseURL = filepath.Join(dbDir, "shared.db")

	sp, err := storage.New(cfg, &storage.Dependencies{Log: &logger})
	require.NoError(t, err)

	replicaA, err := New(cfg, &Dependencies{Log: &logger, StorageProvider: sp})
	require.NoError(t, err)
	replicaB, err := New(cfg, &Dependencies{Log: &logger, StorageProvider: sp})
	require.NoError(t, err)

	// A consumed SAML assertion / client-assertion jti recorded on one replica
	// must be visible on the other, or a replay simply targets the other replica.
	replayKey := "saml_assertion:org-1:" + uuid.New().String()
	require.NoError(t, replicaA.SetCache(replayKey, "1", 600))

	seen, err := replicaB.GetCache(replayKey)
	require.NoError(t, err)
	assert.Equal(t, "1", seen,
		"replica B must observe the assertion replica A already consumed, else replay defence is per-process")

	// Lockout counters must aggregate across replicas, or N replicas grant N x
	// the configured OTP attempts.
	lockKey := "otp_lock:" + uuid.New().String()
	n, err := replicaA.IncrementCache(lockKey, 300)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	n, err = replicaB.IncrementCache(lockKey, 300)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n, "the second attempt must see the first replica's count, not restart at 1")

	n, err = replicaA.IncrementCache(lockKey, 300)
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)

	// Invalidation must clear it everywhere (a correct OTP resets the lockout).
	require.NoError(t, replicaB.DeleteCacheByPrefix(lockKey))
	after, err := replicaA.GetCache(lockKey)
	require.NoError(t, err)
	assert.Empty(t, after, "clearing a lockout on one replica must clear it for all")
}

// TestDBStoreCacheDoesNotCollideWithState pins the separation between the two
// key spaces: a cache write must never be redeemable as an authorization code,
// nor a code readable as a cache entry, even when both use the identical key.
//
// Cache entries live in authorizer_session_tokens under the reserved
// cacheNamespace user_id; codes and SSO state live in authorizer_oauth_states.
// The test does not assume that split — it asserts the observable contract — so
// it keeps holding if the storage layout changes again.
func TestDBStoreCacheDoesNotCollideWithState(t *testing.T) {
	p, err := newTestMemoryStore(t, memoryStoreTypeDB)
	require.NoError(t, err)

	const key = "collision-probe"
	require.NoError(t, p.SetState(key, "the-authorization-code-payload"))
	require.NoError(t, p.SetCache(key, "the-cache-payload", 600))

	cached, err := p.GetCache(key)
	require.NoError(t, err)
	assert.Equal(t, "the-cache-payload", cached, "cache read must not return the state payload")

	state, err := p.GetAndRemoveState(key)
	require.NoError(t, err)
	assert.Equal(t, "the-authorization-code-payload", state, "state read must not return the cache payload")

	// Consuming the code must leave the cache entry intact.
	stillCached, err := p.GetCache(key)
	require.NoError(t, err)
	assert.Equal(t, "the-cache-payload", stillCached, "redeeming a code must not evict a same-named cache entry")
}
