package memory_store

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
	"github.com/authorizerdev/authorizer/internal/memory_store/redis"
)

// Dependencies struct for memory store provider
type Dependencies struct {
	Log *zerolog.Logger
}

// New returns a new memory store provider
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	if cfg.RedisURL != "" {
		return redis.NewRedisProvider(cfg, &redis.Dependencies{
			Log: deps.Log,
		})
	}
	return in_memory.NewInMemoryProvider(&in_memory.Dependencies{
		Log: deps.Log,
	})
}

// Provider defines current memory store provider
type Provider interface {
	// SetUserSession sets the user session for given user identifier in form recipe:user_id
	SetUserSession(userId, key, token string, expiration int64) error
	// GetUserSession returns the session token for given token
	GetUserSession(userId, key string) (string, error)
	// DeleteUserSession deletes the user session
	DeleteUserSession(userId, key string) error
	// DeleteAllSessions deletes all the sessions from the session store
	DeleteAllUserSessions(userId string) error
	// DeleteSessionForNamespace deletes the session for a given namespace
	DeleteSessionForNamespace(namespace string) error
	// SetMfaSession sets the mfa session with key and value of userId
	SetMfaSession(userId, key string, expiration int64) error
	// GetMfaSession returns value of given mfa session
	GetMfaSession(userId, key string) (string, error)
	// GetAllMfaSessions returns all mfa sessions for given userId
	GetAllMfaSessions(userId string) ([]string, error)
	// DeleteMfaSession deletes given mfa session from in-memory store.
	DeleteMfaSession(userId, key string) error

	// SetState sets the login state (key, value form) in the session store
	SetState(key, state string) error
	// GetState returns the state from the session store
	GetState(key string) (string, error)
	// RemoveState removes the social login state from the session store
	RemoveState(key string) error
}
