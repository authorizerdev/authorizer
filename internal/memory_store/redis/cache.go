package redis

import (
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
