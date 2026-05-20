package authorization

import (
	"sync"
	"time"
)

// cache holds in-process membership caches that don't fit the string-only
// memory_store.Provider API. Decision results (allowed / denied for a
// (principal, resource, scope)) live in memory_store instead — see
// evaluator.go.
//
// validSets caches the bounded set of known resource and scope names so
// validateResourceExists / validateScopeExists avoid a storage round-trip
// on every CheckPermission call. A zero-length set is a valid cached value
// meaning "DB was reachable and empty".
type cache struct {
	ttl       time.Duration
	validSets sync.Map // cache key -> map[string]struct{}
	expiryMap sync.Map // cache key -> time.Time
}

// newCache creates a new local membership cache. If ttlSeconds is 0,
// caching is disabled and getValidSet always reports miss.
func newCache(ttlSeconds int64) *cache {
	return &cache{
		ttl: time.Duration(ttlSeconds) * time.Second,
	}
}

// enabled reports whether caching is active (TTL > 0).
func (c *cache) enabled() bool {
	return c.ttl > 0
}

// getValidSet returns the cached membership set for the given key.
// The second return value reports whether the cache had an entry at all.
// Callers must not mutate the returned map.
func (c *cache) getValidSet(key string) (map[string]struct{}, bool) {
	if !c.enabled() {
		return nil, false
	}
	expiry, ok := c.expiryMap.Load(key)
	if !ok {
		return nil, false
	}
	if time.Now().After(expiry.(time.Time)) {
		c.validSets.Delete(key)
		c.expiryMap.Delete(key)
		return nil, false
	}
	v, ok := c.validSets.Load(key)
	if !ok {
		return nil, false
	}
	return v.(map[string]struct{}), true
}

// setValidSet stores a membership set under the given key with the
// configured TTL.
func (c *cache) setValidSet(key string, set map[string]struct{}) {
	if !c.enabled() {
		return
	}
	c.validSets.Store(key, set)
	c.expiryMap.Store(key, time.Now().Add(c.ttl))
}

// invalidateValidSets evicts all cached validSets entries. Called when an
// admin mutation may have changed the resource or scope catalog. No-op when
// caching is disabled — symmetric with setValidSet/getValidSet.
func (c *cache) invalidateValidSets() {
	if !c.enabled() {
		return
	}
	c.validSets.Range(func(key, _ any) bool {
		c.validSets.Delete(key)
		c.expiryMap.Delete(key)
		return true
	})
}

// validResourcesKey returns the cache key for the set of known resource names.
func validResourcesKey() string {
	return "authz:valid_resources"
}

// validScopesKey returns the cache key for the set of known scope names.
func validScopesKey() string {
	return "authz:valid_scopes"
}
