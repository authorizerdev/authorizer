package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// TestDBMemoryStoreProvider tests the database-backed memory store provider
// This test requires a database to be configured
func TestDBMemoryStoreProvider(t *testing.T) {
	// Skip if database is not configured
	cfg := &config.Config{
		DatabaseType: "sqlite",
		DatabaseURL:  "file:test.db?mode=memory&cache=shared",
		Env:          "test",
	}

	// Create storage provider
	storageProvider, err := storage.New(cfg, &storage.Dependencies{})
	if err != nil {
		t.Skipf("Skipping test: failed to create storage provider: %v", err)
		return
	}

	// Create DB memory store provider
	p, err := NewDBProvider(cfg, &Dependencies{
		StorageProvider: storageProvider,
	})
	require.NoError(t, err)
	require.NotNil(t, p)

	// Test SetUserSession and GetUserSession
	err = p.SetUserSession("auth_provider:123", "session_token_key", "test_hash123", time.Now().Add(60*time.Second).Unix())
	assert.NoError(t, err)

	err = p.SetUserSession("auth_provider:123", "access_token_key", "test_jwt123", time.Now().Add(60*time.Second).Unix())
	assert.NoError(t, err)

	// Get session
	key, err := p.GetUserSession("auth_provider:123", "session_token_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_hash123", key)

	key, err = p.GetUserSession("auth_provider:123", "access_token_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_jwt123", key)

	// Test expiration
	err = p.SetUserSession("auth_provider:124", "session_token_key", "test_hash124", time.Now().Add(1*time.Second).Unix())
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)

	key, err = p.GetUserSession("auth_provider:124", "session_token_key")
	assert.Empty(t, key)
	assert.Error(t, err)

	// Test DeleteUserSession
	err = p.DeleteUserSession("auth_provider:123", "session_token_key")
	assert.NoError(t, err)

	key, err = p.GetUserSession("auth_provider:123", "session_token_key")
	assert.Empty(t, key)
	assert.Error(t, err)

	// Test DeleteAllUserSessions
	err = p.SetUserSession("auth_provider:123", "session_token_key1", "test_hash1123", time.Now().Add(60*time.Second).Unix())
	assert.NoError(t, err)

	err = p.DeleteAllUserSessions("123")
	assert.NoError(t, err)

	key, err = p.GetUserSession("auth_provider:123", "session_token_key1")
	assert.Empty(t, key)
	assert.Error(t, err)

	// Test DeleteSessionForNamespace
	err = p.SetUserSession("auth_provider:125", "session_token_key", "test_hash125", time.Now().Add(60*time.Second).Unix())
	assert.NoError(t, err)

	err = p.DeleteSessionForNamespace("auth_provider")
	assert.NoError(t, err)

	key, err = p.GetUserSession("auth_provider:125", "session_token_key")
	assert.Empty(t, key)
	assert.Error(t, err)

	// Test MFA sessions
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

	// Test OAuth state
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
}
