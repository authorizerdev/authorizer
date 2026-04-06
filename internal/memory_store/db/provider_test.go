package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage"
)

// TestDBMemoryStoreProvider tests the database-backed memory store against SQLite.
func TestDBMemoryStoreProvider(t *testing.T) {
	entries := storageTestDBEntriesFromEnv()
	if len(entries) == 0 {
		t.Fatal("no database configurations for memory store DB tests")
	}

	for _, e := range entries {
		t.Run("db="+e.dbType, func(t *testing.T) {
			tempSQLite := filepath.Join(t.TempDir(), "memory_store_test.db")
			dbURL := resolveSQLiteTestURL(e.dbType, e.dbURL, tempSQLite)
			cfg := buildStorageTestConfigForMemoryStore(e.dbType, dbURL)

			log := zerolog.New(zerolog.NewTestWriter(t))
			storageProvider, err := storage.New(cfg, &storage.Dependencies{Log: &log})
			if err != nil {
				t.Skipf("skipping: storage provider for %s: %v", e.dbType, err)
				return
			}

			p, err := NewDBProvider(cfg, &Dependencies{
				Log:             &log,
				StorageProvider: storageProvider,
			})
			require.NoError(t, err)
			require.NotNil(t, p)

			err = p.SetUserSession("auth_provider:123", "session_token_key", "test_hash123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)

			err = p.SetUserSession("auth_provider:123", "access_token_key", "test_jwt123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)

			key, err := p.GetUserSession("auth_provider:123", "session_token_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_hash123", key)

			key, err = p.GetUserSession("auth_provider:123", "access_token_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_jwt123", key)

			err = p.SetUserSession("auth_provider:124", "session_token_key", "test_hash124", time.Now().Add(1*time.Second).Unix())
			assert.NoError(t, err)

			time.Sleep(2 * time.Second)

			key, err = p.GetUserSession("auth_provider:124", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)

			err = p.DeleteUserSession("auth_provider:123", "key")
			assert.NoError(t, err)

			key, err = p.GetUserSession("auth_provider:123", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)

			err = p.SetUserSession("auth_provider:123", "session_token_key1", "test_hash1123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)

			err = p.DeleteAllUserSessions("123")
			assert.NoError(t, err)

			key, err = p.GetUserSession("auth_provider:123", "session_token_key1")
			assert.Empty(t, key)
			assert.Error(t, err)

			err = p.SetUserSession("auth_provider:125", "session_token_key", "test_hash125", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)

			err = p.DeleteSessionForNamespace("auth_provider")
			assert.NoError(t, err)

			key, err = p.GetUserSession("auth_provider:125", "session_token_key")
			assert.Empty(t, key)
			assert.Error(t, err)

			err = p.SetMfaSession("auth_provider:123", "session123", time.Now().Add(60*time.Second).Unix())
			assert.NoError(t, err)

			key, err = p.GetMfaSession("auth_provider:123", "session123")
			assert.NoError(t, err)
			assert.Equal(t, "auth_provider:123", key)

			err = p.DeleteMfaSession("auth_provider:123", "session123")
			assert.NoError(t, err)

			key, err = p.GetMfaSession("auth_provider:123", "session123")
			assert.Error(t, err)
			assert.Empty(t, key)

			err = p.SetState("test_state_key", "test_state_value")
			assert.NoError(t, err)

			state, err := p.GetState("test_state_key")
			assert.NoError(t, err)
			assert.Equal(t, "test_state_value", state)

			err = p.RemoveState("test_state_key")
			assert.NoError(t, err)

			state, err = p.GetState("test_state_key")
			assert.Error(t, err)
			assert.Empty(t, state)
		})
	}
}
