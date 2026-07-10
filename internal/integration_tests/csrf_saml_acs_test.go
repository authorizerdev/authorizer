package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSRFExemptsSAMLACS proves the SAML POST binding survives the CSRF
// middleware: the IdP delivers the assertion via an auto-submitting form from
// ITS OWN origin, so the ACS endpoint must accept a cross-origin POST that the
// Origin allow-list would otherwise reject. The handler defends itself with
// XML signature validation, InResponseTo binding, and the replay cache.
func TestCSRFExemptsSAMLACS(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ts.HttpProvider.CSRFMiddleware())
	reached := false
	router.POST("/oauth/saml/:org_slug/acs", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusBadRequest) // stand-in for SAML validation failure
	})
	router.POST("/oauth/saml/:org_slug/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("cross-origin IdP form POST reaches the ACS handler", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/oauth/saml/acme/acs",
			strings.NewReader("SAMLResponse=dGVzdA%3D%3D&RelayState=abc"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://idp.example.com") // IdP origin, never allow-listed
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.True(t, reached, "ACS handler must be reached, not blocked by CSRF")
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})

	t.Run("other SAML POST paths are NOT exempt", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/oauth/saml/acme/login", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://attacker.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"the ACS exemption must not leak to other /oauth/saml/ POST routes")
	})
}
