package authorization

import (
	"testing"
)

func TestCache_UnmatchedCounter_IncrementsAndReads(t *testing.T) {
	c := newCache(60) // 60s TTL, irrelevant for counter which persists in the separate map
	c.bumpUnmatched("orders", "read")
	c.bumpUnmatched("orders", "read")
	c.bumpUnmatched("users", "delete")

	if got := c.unmatchedCount("orders", "read"); got != 2 {
		t.Fatalf("expected orders:read count=2, got %d", got)
	}
	if got := c.unmatchedCount("users", "delete"); got != 1 {
		t.Fatalf("expected users:delete count=1, got %d", got)
	}
	if got := c.unmatchedCount("nope", "nope"); got != 0 {
		t.Fatalf("expected unknown count=0, got %d", got)
	}
}
