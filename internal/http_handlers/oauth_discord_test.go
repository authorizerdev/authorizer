package http_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/oauth"
)

// newDiscordTestHTTPProvider mirrors newTwitterTestHTTPProvider
// (oauth_twitter_test.go) for Discord.
func newDiscordTestHTTPProvider(t *testing.T, mockBase string) *httpProvider {
	t.Helper()
	config.TestOAuthMockBaseOverride = mockBase
	t.Cleanup(func() { config.TestOAuthMockBaseOverride = "" })
	logger := zerolog.Nop()
	cfg := &config.Config{
		Env:                 constants.E2EEnv,
		DiscordClientID:     "test-client",
		DiscordClientSecret: "test-secret",
	}
	oauthProvider, err := oauth.New(cfg, &oauth.Dependencies{Log: &logger})
	require.NoError(t, err)
	return &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:           &logger,
			OAuthProvider: oauthProvider,
		},
	}
}

// newDiscordTestServer mirrors newTwitterTestServer (oauth_twitter_test.go):
// a local, self-contained mock of mock-oauth's /discord/token and
// /discord/userinfo routes - no live network, no e2e-only config field.
func newDiscordTestServer(t *testing.T, userinfoBody map[string]interface{}) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "mock-access-token",
			"token_type":   "bearer",
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userinfoBody)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// discordProfile builds the flat GET /users/@me shape processDiscordUserInfo
// now expects (id/username/avatar/email at the top level - see the doc
// comment on constants.DiscordUserInfoURL for why this isn't nested under a
// "user" key like the /oauth2/@me endpoint it replaced).
func discordProfile(id, username, avatar, email string) map[string]interface{} {
	profile := map[string]interface{}{
		"id":       id,
		"username": username,
		"avatar":   avatar,
	}
	if email != "" {
		profile["email"] = email
	}
	return profile
}

// TestProcessDiscordUserInfo_RealEmailUsed proves the bug fix: when Discord
// returns a real email (the common case - defaultDiscordScopes already
// requests `email`, cmd/root.go), processDiscordUserInfo sets it as
// user.Email instead of leaving it nil.
func TestProcessDiscordUserInfo_RealEmailUsed(t *testing.T) {
	server := newDiscordTestServer(t, discordProfile("42", "gracehopper", "abc123", "grace@example.com"))
	h := newDiscordTestHTTPProvider(t, server.URL)

	user, err := h.processDiscordUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Equal(t, "grace@example.com", *user.Email)
	assert.NotEqual(t, discordSyntheticEmail("42"), *user.Email)
}

// TestProcessDiscordUserInfo_AbsentEmailFallsBackToSynthetic covers the
// granular-consent edge case: a user can authorize `identify` while denying
// the `email` scope at Discord's consent screen even though the app
// requested it, so /users/@me can still omit email. Falling back to a
// stable per-id synthetic address (mirroring processTwitterUserInfo) keeps
// the signup-vs-login GetUserByEmail lookup working instead of regressing
// to the original bug (nil Email -> GetUserByEmail("") -> duplicate account
// every login).
func TestProcessDiscordUserInfo_AbsentEmailFallsBackToSynthetic(t *testing.T) {
	server := newDiscordTestServer(t, discordProfile("42", "gracehopper", "abc123", ""))
	h := newDiscordTestHTTPProvider(t, server.URL)

	user, err := h.processDiscordUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Equal(t, discordSyntheticEmail("42"), *user.Email)
}

// TestProcessDiscordUserInfo_SameIDYieldsSameEmailAcrossLogins is the
// repeat-login regression guard at the Go level (the e2e counterpart is
// tests/social/discord.spec.ts's "repeat login" test): two separate calls
// for the same Discord id must produce the identical, non-empty email so
// the second login's GetUserByEmail resolves to the first login's account.
func TestProcessDiscordUserInfo_SameIDYieldsSameEmailAcrossLogins(t *testing.T) {
	profile := discordProfile("77", "adalovelace", "def456", "ada@example.com")
	server := newDiscordTestServer(t, profile)
	h := newDiscordTestHTTPProvider(t, server.URL)

	user1, err := h.processDiscordUserInfo(testGinContext(), "code-1")
	require.NoError(t, err)
	user2, err := h.processDiscordUserInfo(testGinContext(), "code-2")
	require.NoError(t, err)

	require.NotNil(t, user1.Email)
	require.NotNil(t, user2.Email)
	assert.Equal(t, *user1.Email, *user2.Email)
}

// TestProcessDiscordUserInfo_MissingID_ReturnsError is the defensive edge
// case mirroring TestProcessTwitterUserInfo_MissingID_ReturnsError: no id
// means no stable anchor for either the avatar URL or the synthetic email
// fallback, so this must error rather than silently minting an
// empty-id-keyed synthetic email that would collide every id-less response
// onto one account.
func TestProcessDiscordUserInfo_MissingID_ReturnsError(t *testing.T) {
	profile := map[string]interface{}{
		"username": "noid",
		"avatar":   "abc123",
	}
	server := newDiscordTestServer(t, profile)
	h := newDiscordTestHTTPProvider(t, server.URL)

	user, err := h.processDiscordUserInfo(testGinContext(), "code")
	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestProcessDiscordUserInfo_GivenNameAndPictureMapping is the regression
// guard on the pre-existing username/avatar mapping this change must not
// disturb.
func TestProcessDiscordUserInfo_GivenNameAndPictureMapping(t *testing.T) {
	server := newDiscordTestServer(t, discordProfile("99", "gracehopper", "xyz789", "grace@example.com"))
	h := newDiscordTestHTTPProvider(t, server.URL)

	user, err := h.processDiscordUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.GivenName)
	require.NotNil(t, user.Picture)
	assert.Equal(t, "gracehopper", *user.GivenName)
	assert.Equal(t, "https://cdn.discordapp.com/avatars/99/xyz789.png", *user.Picture)
}

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
