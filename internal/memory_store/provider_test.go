package memory_store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

const (
	memoryStoreTypeRedis    = "redis"
	memoryStoreTypeInMemory = "inmemory"
	memoryStoreTypeDB       = "db"
)

var memoryStoreTypes = []string{
	memoryStoreTypeRedis,
	memoryStoreTypeInMemory,
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

// TestMemoryStoreProvider tests the memory store provider
func TestMemoryStoreProvider(t *testing.T) {
	for _, storeType := range memoryStoreTypes {
		t.Run("should test memory store provider for "+storeType, func(t *testing.T) {
			cfg := getTestMemoryStorageConfig(storeType)
			p, err := New(cfg, &Dependencies{})
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
		})
	}
}
