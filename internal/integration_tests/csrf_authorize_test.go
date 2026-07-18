package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSRFExemptsAuthorizePost proves POST /authorize (required by RFC 6749
// §3.1 / OIDC Core §3.1.2.1) survives the CSRF middleware: it's reached via
// top-level navigation/form-submit from arbitrary third-party RPs, not a
// cookie-authenticated mutation, so Origin/Content-Type enforcement does not
// apply — same rationale as /userinfo.
func TestCSRFExemptsAuthorizePost(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ts.HttpProvider.CSRFMiddleware())
	reached := false
	router.POST("/authorize", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	router.POST("/other", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("plain form POST reaches the authorize handler", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/authorize", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.True(t, reached, "authorize handler must be reached, not blocked by CSRF")
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})

	t.Run("other POST paths are NOT exempt", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/other", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://attacker.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"the /authorize exemption must not leak to other POST routes")
	})
}
