package memory_store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
)

const (
	memoryStoreTypeRedis    = "redis"
	memoryStoreTypeInMemory = "inmemory"
	memoryStoreTypeDB       = "db"
)

// memoryStoreTypesForTest lists the providers every contract test runs against.
//
// memoryStoreTypeDB is included deliberately: New() selects it for ANY
// deployment that configures a database without REDIS_URL, which makes it the
// default in production — yet it was previously declared here and never
// exercised, so its semantics went unverified. Two real defects lived in that
// gap (a non-atomic GetAndRemoveState that allowed authorization-code replay,
// and a process-local cache that broke SAML/jti replay defence and OTP lockout
// across replicas). It needs no Docker: it runs on SQLite like the rest.
func memoryStoreTypesForTest() []string {
	var types []string
	if redisMemoryStoreTestsEnabled() {
		types = append(types, memoryStoreTypeRedis)
	}
	types = append(types, memoryStoreTypeInMemory, memoryStoreTypeDB)
	return types
}

func getTestMemoryStorageConfig(storageType string) *config.Config {
	cfg := &config.Config{
		Env: "prod",
	}
	switch storageType {
	case memoryStoreTypeRedis:
		cfg.RedisURL = "redis://localhost:6380"
	case memoryStoreTypeInMemory:
		cfg.RedisURL = ""
	case memoryStoreTypeDB:
		cfg.DatabaseType = "sqlite"
		cfg.DatabaseURL = "test.db"
	default:
		cfg.RedisURL = ""
	}
	return cfg
}

// newTestMemoryStore builds a provider of the requested type. The db type is the
// only one needing a StorageProvider — without one New() silently falls through
// to the in-memory store, which is exactly how the db provider avoided coverage.
func newTestMemoryStore(t *testing.T, storeType string) (Provider, error) {
	t.Helper()
	logger := zerolog.Nop()
	cfg := getTestMemoryStorageConfig(storeType)
	deps := &Dependencies{Log: &logger}

	if storeType == memoryStoreTypeDB {
		// Not t.TempDir(): SQLite's WAL sidecars can appear during teardown and
		// make its RemoveAll fail an already-passed test (same reasoning as the
		// integration harness).
		dbDir, err := os.MkdirTemp("", "authorizer-ms-*")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(dbDir) })
		cfg.DatabaseURL = filepath.Join(dbDir, "memory_store_test.db")

		sp, err := storage.New(cfg, &storage.Dependencies{Log: &logger})
		require.NoError(t, err)
		deps.StorageProvider = sp
	}

	return New(cfg, deps)
}

// TestMemoryStoreProvider tests the in-memory provider always; Redis only when TEST_ENABLE_REDIS=1.
// TEST_DBS does not apply (these are not storage-backend tests).
func TestMemoryStoreProvider(t *testing.T) {
	for _, storeType := range memoryStoreTypesForTest() {
		t.Run("should test memory store provider for "+storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			if storeType == memoryStoreTypeRedis && err != nil {
				t.Skipf("skipping redis memory store test (is Redis running on localhost:6380?): %v", err)
			}
			require.NoError(t, err)
			require.NotNil(t, p)
			err = p.SetUserSession("auth_provider:123", "session_token_key", "test_hash123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			err = p.SetUserSession("auth_provider:123", "access_token_key", "test_jwt123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			// Same user multiple session
			err = p.SetUserSession("auth_provider:123", "session_token_key1", "test_hash1123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			err = p.SetUserSession("auth_provider:123", "access_token_key1", "test_jwt1123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			// Different user session
			err = p.SetUserSession("auth_provider:124", "session_token_key", "test_hash124", time.Now().Add(5*time.Second).Unix())
			assert.NoError(t, err)
			err = p.SetUserSession("auth_provider:124", "access_token_key", "test_jwt124", time.Now().Add(5*time.Second).Unix())
			assert.NoError(t, err)
			// Different provider session
			err = p.SetUserSession("auth_provider1:124", "session_token_key", "test_hash124", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			err = p.SetUserSession("auth_provider1:124", "access_token_key", "test_jwt124", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			// Different provider session
			err = p.SetUserSession("auth_provider1:123", "session_token_key", "test_hash1123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			err = p.SetUserSession("auth_provider1:123", "access_token_key", "test_jwt1123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			// Get session
			key, err := p.GetUserSession("auth_provider:123", "session_token_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_hash123", key)
			key, err = p.GetUserSession("auth_provider:123", "access_token_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_jwt123", key)
			key, err = p.GetUserSession("auth_provider:124", "session_token_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_hash124", key)
			key, err = p.GetUserSession("auth_provider:124", "access_token_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_jwt124", key)
			// Expire some tokens and make sure they are empty
			time.Sleep(5 * time.Second)
			key, err = p.GetUserSession("auth_provider:124", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider:124", "access_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			// Delete user session
			err = p.DeleteUserSession("auth_provider:123", "key")
			assert.NoError(t, err)
			err = p.DeleteUserSession("auth_provider:123", "key")
			assert.NoError(t, err)
			key, err = p.GetUserSession("auth_provider:123", "key")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider:123", "access_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			// Delete all user session
			err = p.DeleteAllUserSessions("123")
			assert.NoError(t, err)
			err = p.DeleteAllUserSessions("123")
			assert.NoError(t, err)
			key, err = p.GetUserSession("auth_provider:123", "session_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider:123", "access_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider1:123", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider1:123", "access_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			// Boundary: DeleteAllUserSessions("123") must NOT over-match a
			// different user whose sessions are keyed "<method>:124:...". User
			// 124's still-live session (auth_provider1:124, 60s TTL) must survive —
			// the userID is matched as a colon-bounded segment, not a substring.
			key, err = p.GetUserSession("auth_provider1:124", "session_token_key")
			assert.NoError(t, err, "DeleteAllUserSessions(123) must not delete user 124's session")
			assert.Equal(t, "test_hash124", key)
			// Delete namespace
			err = p.DeleteSessionForNamespace("auth_provider")
			assert.NoError(t, err)
			err = p.DeleteSessionForNamespace("auth_provider1")
			assert.NoError(t, err)
			key, err = p.GetUserSession("auth_provider:123", "session_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider:123", "access_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider1:123", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider1:123", "access_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider:124", "session_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider:124", "access_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider1:124", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)
			key, err = p.GetUserSession("auth_provider1:124", "access_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)

			err = p.SetMfaSession("auth_provider:123", "session123", "test-purpose", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)
			key, err = p.GetMfaSession("auth_provider:123", "session123")
			assert.NoError(t, err)
			assert.Equal(t, "test-purpose", key)

			// GetMfaSessionOwner resolves the owning userID and purpose from a
			// bare session key, without the caller knowing the userID.
			ownerID, ownerPurpose, err := p.GetMfaSessionOwner("session123")
			assert.NoError(t, err)
			assert.Equal(t, "auth_provider:123", ownerID)
			assert.Equal(t, "test-purpose", ownerPurpose)
			// Unknown session key is not found.
			ownerID, ownerPurpose, err = p.GetMfaSessionOwner("does-not-exist")
			assert.Error(t, err)
			assert.Empty(t, ownerID)
			assert.Empty(t, ownerPurpose)

			err = p.DeleteMfaSession("auth_provider:123", "session123")
			assert.NoError(t, err)
			key, err = p.GetMfaSession("auth_provider:123", "session123")
			assert.Error(t, err)
			assert.Empty(t, key)

			// Deleted session is no longer resolvable by owner lookup either.
			ownerID, ownerPurpose, err = p.GetMfaSessionOwner("session123")
			assert.Error(t, err)
			assert.Empty(t, ownerID)
			assert.Empty(t, ownerPurpose)

			// Expired session is not resolvable. Redis rejects a negative TTL
			// on Set, so this in-memory-only check exercises the same
			// expiry-skip path the DB provider test covers deterministically.
			if storeType == memoryStoreTypeInMemory {
				err = p.SetMfaSession("auth_provider:999", "expiredsession", "test-purpose", time.Now().Add(-60*time.Second).Unix())
				assert.NoError(t, err)
				ownerID, ownerPurpose, err = p.GetMfaSessionOwner("expiredsession")
				assert.Error(t, err)
				assert.Empty(t, ownerID)
				assert.Empty(t, ownerPurpose)
			}
		})
	}
}

func redisMemoryStoreTestsEnabled() bool {
	v := strings.TrimSpace(os.Getenv("TEST_ENABLE_REDIS"))
	return v == "1" || strings.EqualFold(v, "true")
}
