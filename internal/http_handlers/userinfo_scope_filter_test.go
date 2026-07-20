package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFilterUserInfoByScopes_OmitsUnsetClaims proves optional claims with no
// value are OMITTED from the userinfo response, not emitted as JSON null or
// an empty string. Regression test for the oidcc-scope-profile conformance
// failure ("gender is not a string with content", etc.) — the validator
// rejects both null and "" as invalid, so an absent key is the only
// spec-compliant way to represent "no value".
func TestFilterUserInfoByScopes_OmitsUnsetClaims(t *testing.T) {
	full := map[string]interface{}{
		"sub":                "user-1",
		"given_name":         "Ada",
		"family_name":        "Lovelace",
		"preferred_username": "ada",
		"gender":             "",          // present but empty — must still be omitted
		"updated_at":         float64(42), // non-string, always kept when present
	}
	scopes := map[string]struct{}{"profile": {}}

	filtered := filterUserInfoByScopes(full, scopes)

	assert.Equal(t, "Ada", filtered["given_name"])
	assert.Equal(t, "Lovelace", filtered["family_name"])
	assert.Equal(t, float64(42), filtered["updated_at"])

	// Never present in `full` at all (no DB column) — must be omitted.
	for _, k := range []string{"name", "profile", "website", "zoneinfo", "locale"} {
		_, ok := filtered[k]
		assert.False(t, ok, "claim %q must be omitted, not present as null", k)
	}
	// Present in `full` but empty/nil — must also be omitted.
	_, ok := filtered["gender"]
	assert.False(t, ok, "empty-string claim must be omitted")
}

func TestFilterUserInfoByScopes_NilValueOmitted(t *testing.T) {
	full := map[string]interface{}{
		"sub":    "user-1",
		"gender": nil,
	}
	scopes := map[string]struct{}{"profile": {}}

	filtered := filterUserInfoByScopes(full, scopes)

	_, ok := filtered["gender"]
	assert.False(t, ok, "nil claim value must be omitted")
}
