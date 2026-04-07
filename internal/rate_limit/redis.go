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
	// Window in seconds = ceil(burst / rps), minimum 1. The previous integer
	// division silently truncated to 0 when burst < rps and produced
	// inconsistent enforcement vs the in-memory limiter (which uses the
	// same effective window via golang.org/x/time/rate). Use ceiling
	// arithmetic so the redis window is at least as long as the rps period.
	window := 1
	if cfg.RateLimitRPS > 0 {
		// ceil(burst / rps) without floats: (a + b - 1) / b
		w := (cfg.RateLimitBurst + cfg.RateLimitRPS - 1) / cfg.RateLimitRPS
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

// Allow checks if a request from the given IP is allowed using Redis.
//
// Errors are PROPAGATED to the caller (the rate-limit middleware) so that
// the operator-controlled RateLimitFailClosed config can actually take
// effect. Previously this returned (true, nil) on any redis error, which
// meant fail-closed mode was a no-op and a flapping redis silently disabled
// rate limiting entirely.
func (p *redisProvider) Allow(ctx context.Context, ip string) (bool, error) {
	key := "rate_limit:" + ip
	result, err := p.client.Eval(ctx, rateLimitScript, []string{key}, p.burst, p.window).Int64()
	if err != nil {
		p.log.Error().Err(err).Str("ip", ip).Msg("rate limit redis error")
		// Default to allowing the request, but PROPAGATE the error so the
		// middleware can fail-closed when configured.
		return true, err
	}
	return result == 1, nil
}

// Close is a no-op for Redis provider (Redis client lifecycle managed elsewhere)
func (p *redisProvider) Close() error {
	return nil
}
