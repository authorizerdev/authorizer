package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscordSyntheticEmail(t *testing.T) {
	assert.Equal(t, discordSyntheticEmail("42"), discordSyntheticEmail("42"), "deterministic for the same id")
	assert.NotEqual(t, discordSyntheticEmail("1"), discordSyntheticEmail("2"), "distinct ids must not collide")
	assert.Contains(t, discordSyntheticEmail("42"), "42")
}

// REGRESSION (account-duplication bug): processDiscordUserInfo used to call
// GET /oauth2/@me, whose user object never includes email regardless of
// granted scopes, and even ignored userRawData["email"] entirely. Every
// Discord login then hit GetUserByEmail(""), which never matches a NULL
// email column in SQL, so every login (even repeat logins from the same
// person) created a brand-new account. resolveDiscordEmail's synthetic
// fallback (keyed on Discord's permanent id, not the mutable username)
// fixes this: the same identity always resolves to the same email, so
// returning users are recognized.
func TestResolveDiscordEmail_NoEmail_FallsBackToSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"username": "gracehopper", "avatar": "abc123"}
	got := resolveDiscordEmail("42", userRawData)
	assert.Equal(t, discordSyntheticEmail("42"), got)
}

// EDGE CASE: a user can authorize `identify` while denying the `email`
// scope at Discord's consent screen even though the app requested it, so
// GET /users/@me can return the field present but empty rather than
// omitting it entirely - must still be treated as absent.
func TestResolveDiscordEmail_EmptyEmail_FallsBackToSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"email": ""}
	got := resolveDiscordEmail("42", userRawData)
	assert.Equal(t, discordSyntheticEmail("42"), got)
}

// GET /users/@me (unlike Twitter/X's confirmed_email) returns a real,
// deliverable email for the common case - defaultDiscordScopes (cmd/root.go)
// already requests `identify email` - and that real address must be
// preferred over the synthetic one.
func TestResolveDiscordEmail_RealEmailPresent_PreferredOverSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"email": "grace@example.com"}
	got := resolveDiscordEmail("42", userRawData)
	assert.Equal(t, "grace@example.com", got)
	assert.NotEqual(t, discordSyntheticEmail("42"), got)
}

// Distinct Discord ids must never collide onto the same synthetic email
// (which would incorrectly merge two real users' accounts).
func TestResolveDiscordEmail_DifferentIDsNoEmail_NeverCollide(t *testing.T) {
	emailA := resolveDiscordEmail("1", map[string]interface{}{})
	emailB := resolveDiscordEmail("2", map[string]interface{}{})
	assert.NotEqual(t, emailA, emailB)
}
