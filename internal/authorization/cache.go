package authorization

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// cache is a local in-memory cache with TTL support.
// It uses sync.Map for concurrent access and tracks per-key expiry.
// A distributed cache (via memory_store) will be layered on top in Phase 7.
type cache struct {
	ttl       time.Duration
	data      sync.Map
	expiryMap sync.Map
	counters  sync.Map // counter key string -> *int64 (atomic-incremented)
	// validSets holds membership-style caches (known resource names, known scope names).
	// Stored separately from .data so string-valued entries never collide, and so
	// the typed map lookup is O(1) without string parsing.
	// A zero-length set is a valid cached value meaning "DB was reachable and empty".
	validSets sync.Map // cache key -> map[string]struct{}
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
// Both positive and negative cached results (authorization "true"/"false")
// follow the same lookup path, avoiding a cache-stampede on repeated
// deny evaluations for the same (principal, resource, scope).
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
// all related cached decisions. Iterates both the string-valued data map and the
// typed validSets map so both storage tiers are wiped in lockstep.
func (c *cache) deleteByPrefix(prefix string) {
	c.data.Range(func(key, _ any) bool {
		if strings.HasPrefix(key.(string), prefix) {
			c.data.Delete(key)
			c.expiryMap.Delete(key)
		}
		return true
	})
	c.validSets.Range(func(key, _ any) bool {
		if strings.HasPrefix(key.(string), prefix) {
			c.validSets.Delete(key)
			c.expiryMap.Delete(key)
		}
		return true
	})
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

// setValidSet stores a membership set under the given key with the configured TTL.
func (c *cache) setValidSet(key string, set map[string]struct{}) {
	if !c.enabled() {
		return
	}
	c.validSets.Store(key, set)
	c.expiryMap.Store(key, time.Now().Add(c.ttl))
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

// unmatchedCounterKey builds the map key for a (resource, scope) unmatched event.
func unmatchedCounterKey(resource, scope string) string {
	return "authz:unmatched:" + resource + ":" + scope
}

// bumpUnmatched increments the unmatched-check counter for the given (resource, scope).
// Counters are in-process only; they are reset on restart. A future dashboard view
// reads them to surface "uncovered checks" to operators during rollout.
func (c *cache) bumpUnmatched(resource, scope string) {
	key := unmatchedCounterKey(resource, scope)
	v, _ := c.counters.LoadOrStore(key, new(int64))
	atomic.AddInt64(v.(*int64), 1)
}

// unmatchedCount returns the current unmatched counter for the given (resource, scope).
// Returns 0 if the key has never been bumped.
func (c *cache) unmatchedCount(resource, scope string) int64 {
	key := unmatchedCounterKey(resource, scope)
	if v, ok := c.counters.Load(key); ok {
		return atomic.LoadInt64(v.(*int64))
	}
	return 0
}
