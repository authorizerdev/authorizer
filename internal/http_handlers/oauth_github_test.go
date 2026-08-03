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

// newGithubTestHTTPProvider mirrors newRobloxTestHTTPProvider
// (oauth_roblox_test.go) for GitHub.
func newGithubTestHTTPProvider(t *testing.T, mockBase string) *httpProvider {
	t.Helper()
	config.TestOAuthMockBaseOverride = mockBase
	t.Cleanup(func() { config.TestOAuthMockBaseOverride = "" })
	logger := zerolog.Nop()
	cfg := &config.Config{
		Env:                constants.E2EEnv,
		GithubClientID:     "test-client",
		GithubClientSecret: "test-secret",
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

// newGithubTestServer mocks the two GitHub endpoints the handler calls:
// /userinfo (GET https://api.github.com/user) and /user/emails.
func newGithubTestServer(t *testing.T, userinfoBody map[string]interface{}, emails []map[string]interface{}) *httptest.Server {
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
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(emails)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// githubProfile is GitHub's real GET /user shape, trimmed but keeping the
// mixed value types the live API always returns: numeric id/counts, boolean
// site_admin, and null for unset optional fields. A response of only string
// values (which an over-simplified mock would produce) never occurs in
// production.
func githubProfile(email interface{}) map[string]interface{} {
	return map[string]interface{}{
		"login":        "ada",
		"id":           583231,
		"node_id":      "MDQ6VXNlcjU4MzIzMQ==",
		"avatar_url":   "https://avatars.githubusercontent.com/u/583231?v=4",
		"name":         "Ada Lovelace",
		"company":      nil,
		"email":        email,
		"hireable":     nil,
		"public_repos": 8,
		"followers":    210,
		"site_admin":   false,
	}
}

// TestProcessGithubUserInfo_MixedTypePayload is the regression guard: the
// handler used to decode GitHub's /user response into a map[string]string,
// which fails outright with "json: cannot unmarshal number into Go value of
// type string" on the numeric id every real response carries - every GitHub
// login broke at the callback.
func TestProcessGithubUserInfo_MixedTypePayload(t *testing.T) {
	server := newGithubTestServer(t, githubProfile("ada@example.com"), nil)
	h := newGithubTestHTTPProvider(t, server.URL)

	user, err := h.processGithubUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Equal(t, "ada@example.com", *user.Email)
	require.NotNil(t, user.GivenName)
	assert.Equal(t, "Ada", *user.GivenName)
	require.NotNil(t, user.FamilyName)
	assert.Equal(t, "Lovelace", *user.FamilyName)
	require.NotNil(t, user.Picture)
	assert.Equal(t, "https://avatars.githubusercontent.com/u/583231?v=4", *user.Picture)
}

// TestProcessGithubUserInfo_NullEmailFallsBackToEmailsEndpoint covers the
// common case: users who keep their email private get `"email": null` on
// /user, so the handler falls back to /user/emails and picks the primary.
func TestProcessGithubUserInfo_NullEmailFallsBackToEmailsEndpoint(t *testing.T) {
	server := newGithubTestServer(t, githubProfile(nil), []map[string]interface{}{
		{"email": "secondary@example.com", "primary": false, "verified": true},
		{"email": "primary@example.com", "primary": true, "verified": true},
	})
	h := newGithubTestHTTPProvider(t, server.URL)

	user, err := h.processGithubUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	require.NotNil(t, user.Email)
	assert.Equal(t, "primary@example.com", *user.Email)
}
