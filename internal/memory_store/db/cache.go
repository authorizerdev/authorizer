package db

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

// cacheStore is a simple in-memory cache used by the DB-backed memory store provider.
// The DB provider delegates session/state storage to the database, but cache
// entries are kept in-memory for performance since they are short-lived and
// tolerant of loss on restart.
var (
	cache      = make(map[string]*cacheEntry)
	cacheMutex sync.RWMutex
)

// SetCache stores a key-value pair with a TTL in seconds.
func (p *provider) SetCache(key string, value string, ttlSeconds int64) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cache[key] = &cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Unix() + ttlSeconds,
	}
	return nil
}

// GetCache retrieves a cached value by key.
// Returns empty string and nil error if the key is not found or expired.
func (p *provider) GetCache(key string) (string, error) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	entry, ok := cache[key]
	if !ok {
		return "", nil
	}
	if entry.ExpiresAt < time.Now().Unix() {
		// Entry expired; clean up asynchronously to avoid write lock in read path.
		go func() {
			cacheMutex.Lock()
			defer cacheMutex.Unlock()
			// Re-check to avoid deleting a refreshed entry.
			if e, exists := cache[key]; exists && e.ExpiresAt < time.Now().Unix() {
				delete(cache, key)
			}
		}()
		return "", nil
	}
	return entry.Value, nil
}

// DeleteCacheByPrefix removes all cache entries whose keys start with the given prefix.
func (p *provider) DeleteCacheByPrefix(prefix string) error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	for k := range cache {
		if strings.HasPrefix(k, prefix) {
			delete(cache, k)
		}
	}
	return nil
}
