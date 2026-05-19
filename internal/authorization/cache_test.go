package authorization

import (
	"testing"
)

func TestCache_ValidSets(t *testing.T) {
	t.Run("miss when caching disabled (TTL=0)", func(t *testing.T) {
		c := newCache(0)
		_, ok := c.getValidSet("authz:valid_resources")
		if ok {
			t.Fatal("expected cache miss when TTL is 0")
		}
	})

	t.Run("miss before any set", func(t *testing.T) {
		c := newCache(60)
		_, ok := c.getValidSet("authz:valid_resources")
		if ok {
			t.Fatal("expected cache miss for unset key")
		}
	})

	t.Run("hit after set", func(t *testing.T) {
		c := newCache(60)
		set := map[string]struct{}{"orders": {}, "users": {}}
		c.setValidSet("authz:valid_resources", set)

		got, ok := c.getValidSet("authz:valid_resources")
		if !ok {
			t.Fatal("expected cache hit after setValidSet")
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(got))
		}
		if _, found := got["orders"]; !found {
			t.Error("expected 'orders' in cached set")
		}
	})

	t.Run("empty set is a valid cache hit", func(t *testing.T) {
		c := newCache(60)
		c.setValidSet("authz:valid_resources", map[string]struct{}{})

		got, ok := c.getValidSet("authz:valid_resources")
		if !ok {
			t.Fatal("expected cache hit for empty set (DB reachable but empty)")
		}
		if len(got) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(got))
		}
	})

	t.Run("invalidateValidSets clears all entries", func(t *testing.T) {
		c := newCache(60)
		c.setValidSet(validResourcesKey(), map[string]struct{}{"orders": {}})
		c.setValidSet(validScopesKey(), map[string]struct{}{"read": {}})

		c.invalidateValidSets()

		if _, ok := c.getValidSet(validResourcesKey()); ok {
			t.Error("expected resources set to be evicted after invalidateValidSets")
		}
		if _, ok := c.getValidSet(validScopesKey()); ok {
			t.Error("expected scopes set to be evicted after invalidateValidSets")
		}
	})

	t.Run("setValidSet is no-op when TTL=0", func(t *testing.T) {
		c := newCache(0)
		c.setValidSet(validResourcesKey(), map[string]struct{}{"orders": {}})
		_, ok := c.getValidSet(validResourcesKey())
		if ok {
			t.Fatal("expected no cache storage when TTL is 0")
		}
	})
}
