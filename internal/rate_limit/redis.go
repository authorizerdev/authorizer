package rate_limit

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Lua script for atomic sliding window rate limiting.
// Returns 1 if allowed, 0 if denied.
// KEYS[1] = rate limit key
// ARGV[1] = max requests (burst)
// ARGV[2] = window size in seconds
var rateLimitScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, window)
end
if current > limit then
    return 0
end
return 1
`

// Compile-time interface guard
var _ Provider = (*redisProvider)(nil)

type redisProvider struct {
	client RedisClient
	burst  int
	window int
	log    *zerolog.Logger
}

func newRedisProvider(cfg *config.Config, deps *Dependencies) (*redisProvider, error) {
	// Window = burst / rps, minimum 1 second
	window := 1
	if cfg.RateLimitRPS > 0 {
		w := int(cfg.RateLimitBurst / cfg.RateLimitRPS)
		if w > 1 {
			window = w
		}
	}
	return &redisProvider{
		client: deps.RedisStore,
		burst:  cfg.RateLimitBurst,
		window: window,
		log:    deps.Log,
	}, nil
}

// Allow checks if a request from the given IP is allowed using Redis
func (p *redisProvider) Allow(ctx context.Context, ip string) (bool, error) {
	key := "rate_limit:" + ip
	result, err := p.client.Eval(ctx, rateLimitScript, []string{key}, p.burst, p.window).Int64()
	if err != nil {
		p.log.Error().Err(err).Str("ip", ip).Msg("rate limit redis error, failing open")
		return true, nil
	}
	return result == 1, nil
}

// Close is a no-op for Redis provider (Redis client lifecycle managed elsewhere)
func (p *redisProvider) Close() error {
	return nil
}
