package authorization

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// cache is a local in-memory cache with TTL support.
// It uses sync.Map for concurrent access and tracks per-key expiry.
// A distributed cache (via memory_store) will be layered on top in Phase 7.
type cache struct {
	ttl       time.Duration
	data      sync.Map
	expiryMap sync.Map
}

// newCache creates a new local cache. If ttlSeconds is 0, caching is disabled.
func newCache(ttlSeconds int64) *cache {
	return &cache{
		ttl: time.Duration(ttlSeconds) * time.Second,
	}
}

// enabled returns true if caching is active (TTL > 0).
func (c *cache) enabled() bool {
	return c.ttl > 0
}

// get retrieves a cached value by key. Returns the value and whether the key
// was found and still valid. Expired entries are lazily deleted on access.
// This returns cached "false" results identically to "true" results,
// ensuring constant-time behavior for both outcomes.
func (c *cache) get(key string) (string, bool) {
	if !c.enabled() {
		return "", false
	}

	expiry, ok := c.expiryMap.Load(key)
	if !ok {
		return "", false
	}
	if time.Now().After(expiry.(time.Time)) {
		// Lazily evict expired entry.
		c.data.Delete(key)
		c.expiryMap.Delete(key)
		return "", false
	}

	val, ok := c.data.Load(key)
	if !ok {
		return "", false
	}
	return val.(string), true
}

// set stores a value in the cache with the configured TTL.
// Both "true" and "false" values are cached (negative caching)
// to prevent cache stampede on non-existent resource:scope combos.
func (c *cache) set(key string, value string) {
	if !c.enabled() {
		return
	}
	c.data.Store(key, value)
	c.expiryMap.Store(key, time.Now().Add(c.ttl))
}

// deleteByPrefix removes all cached entries whose key starts with the given prefix.
// Used when admin mutations change resources, scopes, or policies to invalidate
// all related cached decisions.
func (c *cache) deleteByPrefix(prefix string) {
	c.data.Range(func(key, _ any) bool {
		if strings.HasPrefix(key.(string), prefix) {
			c.data.Delete(key)
			c.expiryMap.Delete(key)
		}
		return true
	})
}

// evalKey constructs a cache key for an authorization evaluation result.
func evalKey(principalID, resource, scope string) string {
	return fmt.Sprintf("authz:eval:%s:%s:%s", principalID, resource, scope)
}

// validResourcesKey returns the cache key for the set of known resource names.
func validResourcesKey() string {
	return "authz:valid_resources"
}

// validScopesKey returns the cache key for the set of known scope names.
func validScopesKey() string {
	return "authz:valid_scopes"
}
