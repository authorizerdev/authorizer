package in_memory

import (
	"strings"
	"sync"
	"time"
)

// cacheEntry holds a cached value with its expiration time.
type cacheEntry struct {
	Value     string
	ExpiresAt int64
}

var cacheStore sync.Map

// SetCache stores a key-value pair with a TTL in seconds.
func (c *provider) SetCache(key string, value string, ttlSeconds int64) error {
	cacheStore.Store(key, &cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Unix() + ttlSeconds,
	})
	return nil
}

// GetCache retrieves a cached value by key.
// Returns empty string and nil error if the key is not found or expired.
func (c *provider) GetCache(key string) (string, error) {
	val, ok := cacheStore.Load(key)
	if !ok {
		return "", nil
	}
	entry := val.(*cacheEntry)
	if entry.ExpiresAt < time.Now().Unix() {
		cacheStore.Delete(key)
		return "", nil
	}
	return entry.Value, nil
}

// DeleteCacheByPrefix removes all cache entries whose keys start with the given prefix.
func (c *provider) DeleteCacheByPrefix(prefix string) error {
	cacheStore.Range(func(key, value any) bool {
		if k, ok := key.(string); ok && strings.HasPrefix(k, prefix) {
			cacheStore.Delete(key)
		}
		return true
	})
	return nil
}
