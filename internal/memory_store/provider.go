package memory_store

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store/db"
	"github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
	"github.com/authorizerdev/authorizer/internal/memory_store/redis"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// Dependencies struct for memory store provider
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

// New returns a new memory store provider
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	if cfg.RedisURL != "" {
		return redis.NewRedisProvider(cfg, &redis.Dependencies{
			Log: deps.Log,
		})
	}
	// If database is configured, use database-backed memory store
	if cfg.DatabaseType != "" && deps.StorageProvider != nil {
		return db.NewDBProvider(cfg, &db.Dependencies{
			Log:             deps.Log,
			StorageProvider: deps.StorageProvider,
		})
	}
	// Fallback to in-memory store
	return in_memory.NewInMemoryProvider(cfg, &in_memory.Dependencies{
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
	// ClaimRefreshToken atomically consumes the refresh-token entry for
	// (userId, nonce) and reports whether THIS call consumed it. It is the
	// single-use gate for refresh-token rotation (OAuth 2.1 §6.1): the token
	// endpoint must win this claim before issuing a replacement, so that under
	// concurrent redemption of the same refresh token exactly one caller
	// proceeds.
	//
	// It exists separately from DeleteUserSession because that method is a
	// best-effort cleanup used by logout/revoke — it removes the session,
	// access and refresh entries for a nonce and nobody cares whether a row was
	// actually there. Here "did I remove it" IS the security decision, so the
	// answer must come from one atomic operation and must concern only the
	// refresh entry. Returns (false, nil) when the entry is already gone.
	ClaimRefreshToken(userId, nonce string) (bool, error)
	// DeleteAllSessions deletes all the sessions from the session store
	DeleteAllUserSessions(userId string) error
	// DeleteSessionForNamespace deletes the session for a given namespace
	DeleteSessionForNamespace(namespace string) error
	// SetMfaSession sets the mfa session, storing purpose as its value so a
	// consumer can tell how the session was obtained (see
	// constants.MFASessionPurpose*).
	SetMfaSession(userId, key, purpose string, expiration int64) error
	// GetMfaSession returns the stored purpose of the given mfa session
	GetMfaSession(userId, key string) (string, error)
	// GetAllMfaSessions returns all mfa sessions for given userId
	GetAllMfaSessions(userId string) ([]string, error)
	// DeleteMfaSession deletes given mfa session from in-memory store.
	DeleteMfaSession(userId, key string) error
	// GetMfaSessionOwner resolves the userID and purpose for a bare mfa session
	// key, without the caller already knowing the owning userID — used when a
	// caller has a valid session cookie but no identifier (e.g. continuing an
	// OAuth-originated MFA challenge, where the frontend never learns the
	// account's email/phone).
	GetMfaSessionOwner(key string) (userID, purpose string, err error)

	// SetState sets the login state (key, value form) in the session store
	SetState(key, state string) error
	// GetState returns the state from the session store
	GetState(key string) (string, error)
	// RemoveState removes the social login state from the session store
	RemoveState(key string) error
	// GetAndRemoveState atomically retrieves and deletes the state entry.
	// Returns the state value and removes it in a single operation to
	// prevent authorization code replay (RFC 6749 §4.1.2).
	GetAndRemoveState(key string) (string, error)

	// SetCache stores a key-value pair with a TTL in seconds.
	// Used by the authorization engine for permission evaluation caching.
	SetCache(key string, value string, ttlSeconds int64) error
	// SetCacheNX stores a key-value pair with a TTL only if the key is not
	// already held, reporting whether THIS call took it. It is the single-use
	// claim primitive for replay defences (SAML assertion IDs, RFC 7523
	// client-assertion jti) — the same role ClaimRefreshToken plays for refresh
	// tokens, but claiming by creation rather than by removal.
	//
	// Implementations MUST decide the race in one operation and MUST never
	// read-then-write: a GetCache/SetCache pair lets two concurrent replays of
	// the same assertion both observe "unseen" and both be accepted.
	//
	// A storage fault returns (false, err) — callers treat that as "not
	// claimed" and reject, so replay defence fails CLOSED.
	SetCacheNX(key string, value string, ttlSeconds int64) (bool, error)
	// GetCache retrieves a cached value by key. Returns empty string and nil error if not found.
	GetCache(key string) (string, error)
	// DeleteCacheByPrefix removes all cache entries whose keys start with the given prefix.
	// Used for cache invalidation when permissions/policies change.
	DeleteCacheByPrefix(prefix string) error
	// IncrementCache atomically increments the integer counter at key (creating
	// it at 1 if absent or expired) and (re)sets its TTL, returning the new
	// value. Unlike a GetCache/SetCache pair, this is safe under concurrent
	// callers - required for any use as a rate-limit/lockout counter, where a
	// non-atomic read-modify-write lets concurrent requests all observe the
	// same pre-increment count and bypass the limit.
	IncrementCache(key string, ttlSeconds int64) (int64, error)

	// GetAllData returns all the data from the session store
	// This is used for testing purposes only
	GetAllData() (map[string]string, error)
}
