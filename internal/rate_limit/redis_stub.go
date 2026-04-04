package rate_limit

import "github.com/authorizerdev/authorizer/internal/config"

func newRedisProvider(cfg *config.Config, deps *Dependencies) (*inMemoryProvider, error) {
	// Stub: will be replaced by redis.go in Task 3
	return newInMemoryProvider(cfg, deps)
}
