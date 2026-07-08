package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIntersectScopes_Attenuation proves the RFC 8693 attenuation core (DC2/H1):
// the effective scope can only NARROW. intersectScopes(requested, ceiling) returns
// exactly the requested scopes that are also in the ceiling — never more.
func TestIntersectScopes_Attenuation(t *testing.T) {
	// Requesting a superset of the ceiling narrows to the ceiling's members only.
	assert.ElementsMatch(t, []string{"read"},
		intersectScopes([]string{"read", "write"}, []string{"read"}),
		"a scope outside the ceiling must be dropped (non-widening)")

	// A scope absent from the ceiling can never be granted — no widening.
	assert.Empty(t, intersectScopes([]string{"admin"}, []string{"read", "write"}),
		"requesting a scope not in the ceiling grants nothing")

	// Empty ceiling is DENY-ALL.
	assert.Empty(t, intersectScopes([]string{"read"}, nil), "empty ceiling denies all")
	assert.Empty(t, intersectScopes([]string{"read"}, []string{}), "empty ceiling denies all")

	// Duplicates are collapsed.
	assert.Equal(t, []string{"read"}, intersectScopes([]string{"read", "read"}, []string{"read"}),
		"duplicate requested scopes are deduplicated")

	// Full overlap is preserved (order follows the requested list).
	assert.Equal(t, []string{"read", "write"},
		intersectScopes([]string{"read", "write"}, []string{"read", "write", "admin"}),
		"scopes within the ceiling are preserved")

	// Re-exchange (requested already ⊆ ceiling) is idempotent — can't grow.
	once := intersectScopes([]string{"read", "write"}, []string{"read", "write", "admin"})
	twice := intersectScopes(once, []string{"read", "write", "admin"})
	assert.Equal(t, once, twice, "re-intersecting an attenuated set cannot re-widen it")
}

// TestActChainDepth proves the delegation-chain depth counter used to enforce the
// hard nesting cap (H1): each nested `act` adds one hop.
func TestActChainDepth(t *testing.T) {
	assert.Equal(t, 0, actChainDepth(nil), "nil chain is depth 0")
	assert.Equal(t, 1, actChainDepth(map[string]interface{}{"sub": "agent1"}),
		"a single immediate actor is depth 1")
	assert.Equal(t, 2, actChainDepth(map[string]interface{}{
		"sub": "agent2",
		"act": map[string]interface{}{"sub": "agent1"},
	}), "one nested actor is depth 2")
	assert.Equal(t, 3, actChainDepth(map[string]interface{}{
		"sub": "agent3",
		"act": map[string]interface{}{
			"sub": "agent2",
			"act": map[string]interface{}{"sub": "agent1"},
		},
	}), "two nested actors is depth 3")
}
