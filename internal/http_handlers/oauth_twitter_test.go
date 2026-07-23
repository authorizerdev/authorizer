package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTwitterSyntheticEmail(t *testing.T) {
	assert.Equal(t, twitterSyntheticEmail("42"), twitterSyntheticEmail("42"), "deterministic for the same id")
	assert.NotEqual(t, twitterSyntheticEmail("1"), twitterSyntheticEmail("2"), "distinct ids must not collide")
	assert.Contains(t, twitterSyntheticEmail("42"), "42")
}

// REGRESSION (account-duplication bug): Twitter's API never returns an
// email by default, so processTwitterUserInfo used to leave user.Email nil.
// OAuthCallbackHandler's signup-vs-login check
// (GetUserByEmail(refs.StringValue(user.Email))) then always ran as
// GetUserByEmail(""), which never matches a NULL email column in SQL - so
// every Twitter login, even for the exact same person, created a brand-new
// account. resolveTwitterEmail's synthetic fallback (keyed on Twitter's
// permanent numeric id, not the mutable username) fixes this: the same
// identity always resolves to the same email, so returning users are
// recognized.
func TestResolveTwitterEmail_NoConfirmedEmail_FallsBackToSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"name": "Ada Lovelace", "username": "ada"}
	got := resolveTwitterEmail("42", userRawData)
	assert.Equal(t, twitterSyntheticEmail("42"), got)
}

// EDGE CASE: X sends the field present but empty (rather than omitting it
// entirely) when a user hasn't confirmed an email - must still be treated
// as absent, not as a real (empty) address.
func TestResolveTwitterEmail_EmptyConfirmedEmail_FallsBackToSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"confirmed_email": ""}
	got := resolveTwitterEmail("42", userRawData)
	assert.Equal(t, twitterSyntheticEmail("42"), got)
}

// REFINEMENT: X API v2 (2025-04-03+) returns confirmed_email when the OAuth
// request carries the users.email scope and the operator's X Developer App
// has "Request email from users" enabled (see TwitterUserInfoURL's doc
// comment in internal/constants/oauth_info_urls.go). When present, the
// real, deliverable email must be preferred over the synthetic one.
func TestResolveTwitterEmail_ConfirmedEmailPresent_PreferredOverSynthetic(t *testing.T) {
	userRawData := map[string]interface{}{"confirmed_email": "ada@example.com"}
	got := resolveTwitterEmail("42", userRawData)
	assert.Equal(t, "ada@example.com", got)
	assert.NotEqual(t, twitterSyntheticEmail("42"), got)
}

// Distinct Twitter ids must never collide onto the same synthetic email
// (which would incorrectly merge two real users' accounts).
func TestResolveTwitterEmail_DifferentIDsNoConfirmedEmail_NeverCollide(t *testing.T) {
	emailA := resolveTwitterEmail("1", map[string]interface{}{})
	emailB := resolveTwitterEmail("2", map[string]interface{}{})
	assert.NotEqual(t, emailA, emailB)
}
