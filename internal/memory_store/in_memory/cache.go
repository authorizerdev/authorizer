package in_memory

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// IncrementCache atomically increments the integer counter at key (creating it
// at 1 if absent or expired) and refreshes its TTL, returning the new value.
// Implemented as an optimistic CompareAndSwap retry loop over the same
// sync.Map SetCache/GetCache already use, so a concurrent increment can never
// observe a stale pre-increment value the way a GetCache+SetCache pair would.
func (c *provider) IncrementCache(key string, ttlSeconds int64) (int64, error) {
	for {
		now := time.Now().Unix()
		old, loaded := cacheStore.Load(key)
		if !loaded {
			entry := &cacheEntry{Value: "1", ExpiresAt: now + ttlSeconds}
			if _, alreadyStored := cacheStore.LoadOrStore(key, entry); !alreadyStored {
				return 1, nil
			}
			continue
		}
		entry := old.(*cacheEntry)
		var current int64
		if entry.ExpiresAt >= now {
			current, _ = strconv.ParseInt(entry.Value, 10, 64)
		}
		next := current + 1
		newEntry := &cacheEntry{Value: strconv.FormatInt(next, 10), ExpiresAt: now + ttlSeconds}
		if cacheStore.CompareAndSwap(key, old, newEntry) {
			return next, nil
		}
	}
}

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
