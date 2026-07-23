package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRobloxSyntheticEmail(t *testing.T) {
	assert.Equal(t, robloxSyntheticEmail("42"), robloxSyntheticEmail("42"), "deterministic for the same id")
	assert.NotEqual(t, robloxSyntheticEmail("1"), robloxSyntheticEmail("2"), "distinct ids must not collide")
	assert.Contains(t, robloxSyntheticEmail("42"), "42")
}

// REGRESSION (data-quality bug): defaultRobloxScopes (cmd/root.go) is
// ["openid", "profile"] - no `email` scope - so real Roblox userinfo (an
// OIDC-standard endpoint) omits `email` under the default config.
// resolveRobloxEmail used to fall back to the bare numeric `sub`, storing a
// non-email-shaped value directly in user.Email. The fix synthesizes a
// stable, email-shaped address from `sub` instead, mirroring
// resolveTwitterEmail/resolveDiscordEmail.
func TestResolveRobloxEmail_NoEmail_FallsBackToSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"name": "Ada Lovelace", "nickname": "ada"}
	got := resolveRobloxEmail("123456789", userRawData)
	assert.Equal(t, robloxSyntheticEmail("123456789"), got)
	assert.NotEqual(t, "123456789", got, "the raw sub must never land directly in the result")
	assert.Contains(t, got, "@", "fallback must be email-shaped")
}

// EDGE CASE: the response can carry the field present but empty rather than
// omitting it entirely - must still be treated as absent.
func TestResolveRobloxEmail_EmptyEmail_FallsBackToSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"email": ""}
	got := resolveRobloxEmail("42", userRawData)
	assert.Equal(t, robloxSyntheticEmail("42"), got)
}

// When the operator has opted into the `email` scope, the real address must
// be preferred over the synthetic one.
func TestResolveRobloxEmail_RealEmailPresent_PreferredOverSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"email": "ada@example.com"}
	got := resolveRobloxEmail("42", userRawData)
	assert.Equal(t, "ada@example.com", got)
	assert.NotEqual(t, robloxSyntheticEmail("42"), got)
}

// Distinct Roblox subs must never collide onto the same synthetic email
// (which would incorrectly merge two real users' accounts).
func TestResolveRobloxEmail_DifferentSubsNoEmail_NeverCollide(t *testing.T) {
	emailA := resolveRobloxEmail("1", map[string]interface{}{})
	emailB := resolveRobloxEmail("2", map[string]interface{}{})
	assert.NotEqual(t, emailA, emailB)
}

// DEFENSIVE EDGE CASE: an empty-string sub (present key, zero value) must be
// treated the same as a missing sub, not passed to robloxSyntheticEmail -
// which would otherwise mint the same collision-prone
// "roblox-@roblox.oauth.internal" address for every such response.
func TestResolveRobloxEmail_EmptySub_LeavesEmailEmpty(t *testing.T) {
	got := resolveRobloxEmail("", map[string]interface{}{})
	assert.Empty(t, got)
}
