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
	"github.com/authorizerdev/authorizer/internal/oauth"
)

// newRobloxTestHTTPProvider mirrors newTwitterTestHTTPProvider
// (oauth_twitter_test.go) for Roblox.
func newRobloxTestHTTPProvider(t *testing.T, mockBase string) *httpProvider {
	t.Helper()
	logger := zerolog.Nop()
	cfg := &config.Config{
		RobloxClientID:         "test-client",
		RobloxClientSecret:     "test-secret",
		TestOAuthRobloxBaseURL: mockBase,
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

// newRobloxTestServer mirrors newTwitterTestServer (oauth_twitter_test.go): a
// local, self-contained mock of mock-oauth's /roblox/token and
// /roblox/userinfo routes - no live network, no e2e-only config field.
func newRobloxTestServer(t *testing.T, userinfoBody map[string]interface{}) *httptest.Server {
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

// robloxProfile builds Roblox's real OIDC userinfo shape
// (constants.RobloxUserInfoURL - an OIDC-standard endpoint, so `sub` is
// always present regardless of granted scopes; `email` is only included
// once the `email` scope is granted, which defaultRobloxScopes, cmd/root.go,
// does not request by default).
func robloxProfile(sub, name, nickname, email string) map[string]interface{} {
	profile := map[string]interface{}{
		"sub":      sub,
		"name":     name,
		"nickname": nickname,
		"picture":  "https://example.com/a.png",
	}
	if email != "" {
		profile["email"] = email
	}
	return profile
}

// TestProcessRobloxUserInfo_RealEmailUsed is the regression guard for the
// pre-existing, still-correct path: when `email` is present (operator opted
// into the `email` scope), it's used as-is.
func TestProcessRobloxUserInfo_RealEmailUsed(t *testing.T) {
	server := newRobloxTestServer(t, robloxProfile("42", "Ada Lovelace", "ada", "ada@example.com"))
	h := newRobloxTestHTTPProvider(t, server.URL)

	user, err := h.processRobloxUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Equal(t, "ada@example.com", *user.Email)
	assert.NotEqual(t, robloxSyntheticEmail("42"), *user.Email)
}

// TestProcessRobloxUserInfo_AbsentEmailFallsBackToSynthetic proves the bug
// fix: defaultRobloxScopes (cmd/root.go) is ["openid", "profile"] - no
// `email` - so real Roblox userinfo omits `email` under the default config.
// Before the fix, processRobloxUserInfo stuffed the bare numeric `sub` into
// user.Email (not email-shaped at all); the fix synthesizes a stable,
// email-shaped address from `sub` instead, mirroring
// twitterSyntheticEmail/discordSyntheticEmail.
func TestProcessRobloxUserInfo_AbsentEmailFallsBackToSynthetic(t *testing.T) {
	server := newRobloxTestServer(t, robloxProfile("123456789", "Ada Lovelace", "ada", ""))
	h := newRobloxTestHTTPProvider(t, server.URL)

	user, err := h.processRobloxUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Equal(t, robloxSyntheticEmail("123456789"), *user.Email)
	assert.NotEqual(t, "123456789", *user.Email, "the raw sub must never land directly in user.Email")
	assert.Contains(t, *user.Email, "@", "fallback must be email-shaped")
}

// TestProcessRobloxUserInfo_SameIDYieldsSameEmailAcrossLogins is the
// repeat-login regression guard: two separate calls for the same Roblox sub
// must produce the identical, non-empty email so a second login resolves to
// the first login's account instead of creating a duplicate.
func TestProcessRobloxUserInfo_SameIDYieldsSameEmailAcrossLogins(t *testing.T) {
	profile := robloxProfile("77", "Grace Hopper", "grace", "")
	server := newRobloxTestServer(t, profile)
	h := newRobloxTestHTTPProvider(t, server.URL)

	user1, err := h.processRobloxUserInfo(testGinContext(), "code-1")
	require.NoError(t, err)
	user2, err := h.processRobloxUserInfo(testGinContext(), "code-2")
	require.NoError(t, err)

	require.NotNil(t, user1.Email)
	require.NotNil(t, user2.Email)
	assert.Equal(t, *user1.Email, *user2.Email)
}

// TestProcessRobloxUserInfo_MissingSubAndEmail_LeavesEmailEmpty covers the
// defensive edge case where the response has neither `email` nor `sub` -
// there is no anchor to synthesize a stable address from, so email stays
// empty rather than colliding every such response onto one synthetic
// account. Unlike Twitter/Discord (which error on a missing id because
// their id is also load-bearing for other fields, e.g. Discord's avatar
// URL), Roblox has no other use for `sub`, so this simply leaves Email
// empty rather than erroring the whole login.
func TestProcessRobloxUserInfo_MissingSubAndEmail_LeavesEmailEmpty(t *testing.T) {
	profile := map[string]interface{}{
		"name":     "No Sub User",
		"nickname": "nosub",
	}
	server := newRobloxTestServer(t, profile)
	h := newRobloxTestHTTPProvider(t, server.URL)

	user, err := h.processRobloxUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Empty(t, *user.Email)
}

// TestProcessRobloxUserInfo_EmptySubAndAbsentEmail_LeavesEmailEmpty covers
// the case the sub != "" guard exists for: an empty-string `sub` (present
// key, zero value) must be treated the same as a missing `sub`, not passed
// to robloxSyntheticEmail - which would otherwise mint the same collision-
// prone "roblox-@roblox.oauth.internal" address for every such response.
func TestProcessRobloxUserInfo_EmptySubAndAbsentEmail_LeavesEmailEmpty(t *testing.T) {
	profile := map[string]interface{}{
		"sub":      "",
		"name":     "Empty Sub User",
		"nickname": "emptysub",
	}
	server := newRobloxTestServer(t, profile)
	h := newRobloxTestHTTPProvider(t, server.URL)

	user, err := h.processRobloxUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Empty(t, *user.Email)
}

// TestProcessRobloxUserInfo_NameGivenFamilyNicknameMapping is a regression
// guard on the pre-existing name-splitting/nickname/picture mapping this
// change must not disturb: like processTwitterUserInfo,
// processRobloxUserInfo uses strings.SplitAfterN, which keeps the separator
// on the first piece, so "Ada Lovelace" splits to given_name "Ada "
// (trailing space) and family_name "Lovelace" - not GitHub's clean
// strings.Split-based "Ada"/"Lovelace".
func TestProcessRobloxUserInfo_NameGivenFamilyNicknameMapping(t *testing.T) {
	server := newRobloxTestServer(t, robloxProfile("99", "Ada Lovelace", "ada99", "ada@example.com"))
	h := newRobloxTestHTTPProvider(t, server.URL)

	user, err := h.processRobloxUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.GivenName)
	require.NotNil(t, user.FamilyName)
	require.NotNil(t, user.Nickname)
	require.NotNil(t, user.Picture)
	assert.Equal(t, "Ada ", *user.GivenName)
	assert.Equal(t, "Lovelace", *user.FamilyName)
	assert.Equal(t, "ada99", *user.Nickname)
	assert.Equal(t, "https://example.com/a.png", *user.Picture)
}

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
