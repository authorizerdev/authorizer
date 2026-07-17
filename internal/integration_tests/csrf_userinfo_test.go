package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSRFExemptsUserInfoPost proves POST /userinfo (required by OIDC Core
// §5.3.1) survives the CSRF middleware: it's authenticated via a bearer
// access token, not cookies, so Origin allow-listing does not apply — same
// rationale as /oauth/token.
func TestCSRFExemptsUserInfoPost(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ts.HttpProvider.CSRFMiddleware())
	reached := false
	router.POST("/userinfo", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	router.POST("/other", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("bearer-authenticated POST reaches the userinfo handler", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/userinfo", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer sometoken")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.True(t, reached, "userinfo handler must be reached, not blocked by CSRF")
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})

	t.Run("other POST paths are NOT exempt", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/other", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://attacker.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"the /userinfo exemption must not leak to other POST routes")
	})
}
