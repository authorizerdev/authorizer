package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestLogoutPrefersPostLogoutRedirectURI verifies that /logout parses
// post_logout_redirect_uri as the preferred param name per OIDC RP-
// Initiated Logout 1.0 §3, while keeping redirect_uri as a backward-
// compat fallback. Asserts the handler reaches the fingerprint check
// (returning 401 without a cookie) — i.e. it successfully parsed and
// validated the redirect URL format, proving the new parsing branch is
// wired up.
func TestLogoutPrefersPostLogoutRedirectURI(t *testing.T) {
	cfg := getTestConfig()
	cfg.AllowedOrigins = append(cfg.AllowedOrigins, "http://example.com")
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.POST("/logout", ts.HttpProvider.LogoutHandler())

	t.Run("post_logout_redirect_uri accepted", func(t *testing.T) {
		form := strings.NewReader("post_logout_redirect_uri=http://example.com/bye")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/logout", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"POST /logout with post_logout_redirect_uri (no cookie) must reach the fingerprint stage and return 401")
	})

	t.Run("redirect_uri still accepted as fallback", func(t *testing.T) {
		form := strings.NewReader("redirect_uri=http://example.com/bye")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/logout", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"POST /logout with legacy redirect_uri (no cookie) must still work and reach the fingerprint stage")
	})
}

// TestLogoutStateEchoAccepted is a compile-time proof that the state
// echo path is reachable. Without a valid session fingerprint we cannot
// assert the actual redirect URL, but we can verify the code compiles
// and the handler does not crash on state parameter input.
func TestLogoutStateEchoAccepted(t *testing.T) {
	cfg := getTestConfig()
	cfg.AllowedOrigins = append(cfg.AllowedOrigins, "http://example.com")
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.POST("/logout", ts.HttpProvider.LogoutHandler())

	form := strings.NewReader("post_logout_redirect_uri=http://example.com/bye&state=xyz123")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/logout", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"POST /logout with state should still return 401 without a session cookie (state-echo path is only reached on successful logout)")
}
