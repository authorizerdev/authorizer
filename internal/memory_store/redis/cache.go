package redis

import (
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const cachePrefix = "authorizer_cache:"

// SetCache stores a key-value pair in Redis with a TTL in seconds.
func (p *provider) SetCache(key string, value string, ttlSeconds int64) error {
	duration := time.Duration(ttlSeconds) * time.Second
	err := p.store.Set(p.ctx, cachePrefix+key, value, duration).Err()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error setting cache in redis")
		return err
	}
	return nil
}

// GetCache retrieves a cached value by key from Redis.
// Returns empty string and nil error if the key is not found.
func (p *provider) GetCache(key string) (string, error) {
	data, err := p.store.Get(p.ctx, cachePrefix+key).Result()
	if err != nil {
		if err == goredis.Nil {
			return "", nil
		}
		p.dependencies.Log.Debug().Err(err).Msg("Error getting cache from redis")
		return "", err
	}
	return data, nil
}

// IncrementCache atomically increments the integer counter at key (creating it
// at 1 if absent) and refreshes its TTL, returning the new value. INCR+EXPIRE
// run as a single Lua script (matching the GetAndRemoveState pattern below) so
// concurrent callers can never observe the same pre-increment value the way a
// GET+SET pair would.
func (p *provider) IncrementCache(key string, ttlSeconds int64) (int64, error) {
	fullKey := cachePrefix + key
	script := `local v = redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], ARGV[1])
return v`
	result, err := p.store.Eval(p.ctx, script, []string{fullKey}, ttlSeconds).Result()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error incrementing cache in redis")
		return 0, err
	}
	next, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected redis reply type for IncrementCache")
	}
	return next, nil
}

// DeleteCacheByPrefix removes all cache entries whose keys start with the given prefix.
// Uses SCAN to avoid blocking Redis on large datasets.
func (p *provider) DeleteCacheByPrefix(prefix string) error {
	pattern := cachePrefix + prefix + "*"
	var cursor uint64
	for {
		keys, nextCursor, err := p.store.Scan(p.ctx, cursor, pattern, 100).Result()
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Error scanning cache keys from redis")
			return err
		}
		if len(keys) > 0 {
			if err := p.store.Del(p.ctx, keys...).Err(); err != nil {
				p.dependencies.Log.Debug().Err(err).Msg("Error deleting cache keys from redis")
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
