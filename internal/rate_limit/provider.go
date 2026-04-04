package rate_limit

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// RedisClient is any Redis client that supports Eval (e.g., *redis.Client, *redis.ClusterClient)
type RedisClient = redis.Cmdable

// Provider defines the rate limiting interface
type Provider interface {
	// Allow checks if a request from the given IP should be allowed
	Allow(ctx context.Context, ip string) (bool, error)
	// Close cleans up resources used by the provider
	Close() error
}

// Dependencies for rate limit provider
type Dependencies struct {
	Log        *zerolog.Logger
	RedisStore RedisClient
}

// New creates a new rate limit provider based on available infrastructure.
// Uses Redis when RedisStore is provided, falls back to in-memory.
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	if deps.RedisStore != nil {
		return newRedisProvider(cfg, deps)
	}
	return newInMemoryProvider(cfg, deps)
}
