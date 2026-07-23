package http_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/oauth"
)

// newTwitterTestHTTPProvider builds a minimal httpProvider wired to a mock
// Twitter token+userinfo server, mirroring the TestOAuth*BaseURL override
// e2e-playground/docker-compose.yml uses in production (see
// internal/config/test_oauth_override.go and internal/oauth/get_oauth_config.go).
func newTwitterTestHTTPProvider(t *testing.T, mockBase string) *httpProvider {
	t.Helper()
	logger := zerolog.Nop()
	cfg := &config.Config{
		TwitterClientID:         "test-client",
		TwitterClientSecret:     "test-secret",
		TestOAuthTwitterBaseURL: mockBase,
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

// newTwitterTestServer starts a mock Twitter token+userinfo endpoint
// (mirroring e2e-playground/mocks/mock-oauth/server.ts's /twitter/token and
// /twitter/userinfo routes) that always returns userinfoBody, regardless of
// the exchanged code.
func newTwitterTestServer(t *testing.T, userinfoBody map[string]interface{}) *httptest.Server {
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

func testGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "http://localhost/oauth_callback/twitter", nil)
	return ctx
}

func twitterProfile(id, name, username string) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"id":                id,
			"name":              name,
			"username":          username,
			"profile_image_url": "https://example.com/a.png",
		},
	}
}

// TestProcessTwitterUserInfo_SameIDYieldsSameSyntheticEmail proves the bug
// fix: two separate calls to processTwitterUserInfo for the same Twitter
// account (same numeric id) produce the identical, non-empty synthetic
// email. Since OAuthCallbackHandler's signup-vs-login branch is
// GetUserByEmail(refs.StringValue(user.Email)), a stable non-empty email is
// exactly what lets the second login resolve to the first login's account
// instead of GetUserByEmail("") always missing (the original bug: Email was
// always nil for Twitter).
func TestProcessTwitterUserInfo_SameIDYieldsSameSyntheticEmail(t *testing.T) {
	server := newTwitterTestServer(t, twitterProfile("42", "Ada Lovelace", "ada"))
	h := newTwitterTestHTTPProvider(t, server.URL)

	user1, err := h.processTwitterUserInfo(testGinContext(), "code-1", "verifier-1")
	require.NoError(t, err)
	user2, err := h.processTwitterUserInfo(testGinContext(), "code-2", "verifier-2")
	require.NoError(t, err)

	require.NotNil(t, user1.Email)
	require.NotNil(t, user2.Email)
	assert.NotEmpty(t, *user1.Email)
	assert.Equal(t, *user1.Email, *user2.Email, "same Twitter id must yield the same synthetic email across logins")
}

// TestProcessTwitterUserInfo_DifferentIDsYieldDifferentEmails proves the fix
// doesn't over-match: two distinct Twitter accounts must never collide onto
// the same synthetic email (which would incorrectly merge two real users'
// accounts).
func TestProcessTwitterUserInfo_DifferentIDsYieldDifferentEmails(t *testing.T) {
	serverA := newTwitterTestServer(t, twitterProfile("1", "Alice", "alice"))
	serverB := newTwitterTestServer(t, twitterProfile("2", "Bob", "bob"))

	userA, err := newTwitterTestHTTPProvider(t, serverA.URL).processTwitterUserInfo(testGinContext(), "code-a", "verifier-a")
	require.NoError(t, err)
	userB, err := newTwitterTestHTTPProvider(t, serverB.URL).processTwitterUserInfo(testGinContext(), "code-b", "verifier-b")
	require.NoError(t, err)

	require.NotNil(t, userA.Email)
	require.NotNil(t, userB.Email)
	assert.NotEqual(t, *userA.Email, *userB.Email, "different Twitter ids must never produce the same synthetic email")
}

// TestProcessTwitterUserInfo_MissingID_ReturnsError proves the defensive
// edge case: if Twitter's response ever omits `id` (not expected per the
// real API contract, but not to be trusted blindly either), the function
// errors out rather than silently minting a synthetic email keyed on an
// empty id (which would collide every id-less response onto one account).
func TestProcessTwitterUserInfo_MissingID_ReturnsError(t *testing.T) {
	profile := map[string]interface{}{
		"data": map[string]interface{}{
			"name":     "No Id User",
			"username": "noid",
		},
	}
	server := newTwitterTestServer(t, profile)
	h := newTwitterTestHTTPProvider(t, server.URL)

	user, err := h.processTwitterUserInfo(testGinContext(), "code", "verifier")
	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestProcessTwitterUserInfo_NameGivenFamilyNicknameMapping is a regression
// guard on the pre-existing name-splitting/nickname/picture mapping this
// change must not disturb: strings.SplitAfterN keeps the separator on the
// first piece, so "Ada Lovelace" splits to given_name "Ada " (trailing
// space) and family_name "Lovelace", exactly as before this fix.
func TestProcessTwitterUserInfo_NameGivenFamilyNicknameMapping(t *testing.T) {
	server := newTwitterTestServer(t, twitterProfile("99", "Ada Lovelace", "ada99"))
	h := newTwitterTestHTTPProvider(t, server.URL)

	user, err := h.processTwitterUserInfo(testGinContext(), "code", "verifier")
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

func TestTwitterSyntheticEmail(t *testing.T) {
	assert.Equal(t, twitterSyntheticEmail("42"), twitterSyntheticEmail("42"), "deterministic for the same id")
	assert.NotEqual(t, twitterSyntheticEmail("1"), twitterSyntheticEmail("2"), "distinct ids must not collide")
	assert.Contains(t, twitterSyntheticEmail("42"), "42")
}
