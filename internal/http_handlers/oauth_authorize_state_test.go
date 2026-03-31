package http_handlers

import (
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	inmemorystore "github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
)

func TestConsumeAuthorizeState_Nonce(t *testing.T) {
	cfg := &config.Config{}
	logger := zerolog.Nop()
	ms, err := inmemorystore.NewInMemoryProvider(cfg, &inmemorystore.Dependencies{Log: &logger})
	require.NoError(t, err)

	h := &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:               &logger,
			MemoryStoreProvider: ms,
		},
	}

	stateValue := "state-1"
	require.NoError(t, ms.SetState(stateValue, "nonce-123"))

	code, codeChallenge, nonce, err := h.consumeAuthorizeState(stateValue)
	require.NoError(t, err)
	require.Empty(t, code)
	require.Empty(t, codeChallenge)
	require.Equal(t, "nonce-123", nonce)

	// Consumed
	after, err := ms.GetState(stateValue)
	require.NoError(t, err)
	require.Empty(t, after)
}

func TestConsumeAuthorizeState_CodeAndPKCE(t *testing.T) {
	cfg := &config.Config{}
	logger := zerolog.Nop()
	ms, err := inmemorystore.NewInMemoryProvider(cfg, &inmemorystore.Dependencies{Log: &logger})
	require.NoError(t, err)

	h := &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:               &logger,
			MemoryStoreProvider: ms,
		},
	}

	stateValue := "state-2"
	require.NoError(t, ms.SetState(stateValue, "code-abc@@challenge-xyz"))

	code, codeChallenge, nonce, err := h.consumeAuthorizeState(stateValue)
	require.NoError(t, err)
	require.Equal(t, "code-abc", code)
	require.Equal(t, "challenge-xyz", codeChallenge)
	require.Empty(t, nonce)

	// Consumed
	after, err := ms.GetState(stateValue)
	require.NoError(t, err)
	require.Empty(t, after)
}

func TestConsumeAuthorizeState_MissingKey_ReturnsEmpty(t *testing.T) {
	cfg := &config.Config{}
	logger := zerolog.Nop()
	ms, err := inmemorystore.NewInMemoryProvider(cfg, &inmemorystore.Dependencies{Log: &logger})
	require.NoError(t, err)

	h := &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:               &logger,
			MemoryStoreProvider: ms,
		},
	}

	code, codeChallenge, nonce, err := h.consumeAuthorizeState("does-not-exist")
	require.NoError(t, err)
	require.Empty(t, code)
	require.Empty(t, codeChallenge)
	require.Empty(t, nonce)
}

// This models Redis behaviour where missing keys return redis.Nil.
func TestConsumeAuthorizeState_RedisNil_Propagates(t *testing.T) {
	cfg := &config.Config{}
	logger := zerolog.Nop()

	h := &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log: &logger,
			MemoryStoreProvider: &fakeMemoryStore{
				getStateErr: goredis.Nil,
			},
		},
	}

	_, _, _, err := h.consumeAuthorizeState("missing")
	require.ErrorIs(t, err, goredis.Nil)
}

type fakeMemoryStore struct {
	getStateVal string
	getStateErr error

	removedKeys []string
}

func (f *fakeMemoryStore) SetUserSession(userId, key, token string, expiration int64) error { return nil }
func (f *fakeMemoryStore) GetUserSession(userId, key string) (string, error)                { return "", nil }
func (f *fakeMemoryStore) DeleteUserSession(userId, key string) error                       { return nil }
func (f *fakeMemoryStore) DeleteAllUserSessions(userId string) error                        { return nil }
func (f *fakeMemoryStore) DeleteSessionForNamespace(namespace string) error                 { return nil }
func (f *fakeMemoryStore) SetMfaSession(userId, key string, expiration int64) error         { return nil }
func (f *fakeMemoryStore) GetMfaSession(userId, key string) (string, error)                 { return "", nil }
func (f *fakeMemoryStore) GetAllMfaSessions(userId string) ([]string, error)                { return nil, nil }
func (f *fakeMemoryStore) DeleteMfaSession(userId, key string) error                        { return nil }
func (f *fakeMemoryStore) SetState(key, state string) error                                 { return nil }
func (f *fakeMemoryStore) GetState(key string) (string, error)                              { return f.getStateVal, f.getStateErr }
func (f *fakeMemoryStore) RemoveState(key string) error {
	f.removedKeys = append(f.removedKeys, key)
	return nil
}
func (f *fakeMemoryStore) GetAllData() (map[string]string, error) { return map[string]string{}, nil }

