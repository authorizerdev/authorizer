package memory_store

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
)

// TestRedisClient_ReachesRateLimiter: with Redis configured, the rate limiter
// must receive the memory store's Redis client so limits are shared across
// replicas. It never did — cmd/root.go type-asserted against an interface whose
// Client() return type could not match — and the memory-store errorBoundary
// would hide Client() as well. Every instance silently fell back to a
// per-process in-memory limiter.
func TestRedisClient_ReachesRateLimiter(t *testing.T) {
	if !redisMemoryStoreTestsEnabled() {
		t.Skip("set TEST_ENABLE_REDIS=1 to run Redis memory_store tests")
	}
	p, err := newTestMemoryStore(t, memoryStoreTypeRedis)
	require.NoError(t, err)

	client := RedisClient(p)
	require.NotNil(t, client, "a Redis-backed memory store must expose its client through the error boundary")

	log := zerolog.Nop()
	rl, err := rate_limit.New(&config.Config{RateLimitRPS: 10, RateLimitBurst: 2}, &rate_limit.Dependencies{Log: &log, RedisStore: client})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rl.Close() })
	assert.Contains(t, fmt.Sprintf("%T", rl), "redisProvider")

	// The limiter really talks to Redis: its Eval-based check succeeds.
	allowed, err := rl.Allow(context.Background(), "203.0.113.7")
	require.NoError(t, err)
	assert.True(t, allowed)
}

// TestRedisClient_NilWithoutRedis: the in-memory and DB-backed stores have no
// Redis client, so the rate limiter keeps its in-memory fallback.
func TestRedisClient_NilWithoutRedis(t *testing.T) {
	for _, storeType := range []string{memoryStoreTypeInMemory, memoryStoreTypeDB} {
		t.Run(storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			require.NoError(t, err)
			assert.Nil(t, RedisClient(p))
		})
	}
}
