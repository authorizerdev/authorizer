package memory_store

import (
	goredis "github.com/redis/go-redis/v9"

	"github.com/authorizerdev/authorizer/internal/memory_store/redis"
)

// RedisClient returns the Redis client behind a Redis-backed memory store, so
// other subsystems (the distributed rate limiter) can share the connection. It
// returns nil for the in-memory and DB-backed stores.
//
// It looks through the errorBoundary New adds, and asserts against the
// backend's real method (Client() redis.RedisClient). The previous wiring in
// cmd/root.go asserted an anonymous interface whose Client() return type could
// never match that signature, so the rate limiter silently ran per-process
// in-memory on every deployment.
func RedisClient(p Provider) goredis.Cmdable {
	if b, ok := p.(*errorBoundary); ok {
		p = b.next
	}
	cp, ok := p.(interface{ Client() redis.RedisClient })
	if !ok {
		return nil
	}
	// The concrete client is *goredis.Client or *goredis.ClusterClient, both
	// of which implement the full Cmdable the rate limiter takes.
	client, ok := cp.Client().(goredis.Cmdable)
	if !ok {
		return nil
	}
	return client
}
