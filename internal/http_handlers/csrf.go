package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSRFMiddleware protects against CSRF by requiring state-changing requests
// (POST, PUT, DELETE, PATCH) to include a custom header that browsers will
// not send cross-origin without a CORS preflight.
// OAuth callback POST routes are exempt as they originate from provider redirects.
func (h *httpProvider) CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Exempt OAuth callback routes (provider POST redirects)
		if strings.HasPrefix(c.Request.URL.Path, "/oauth_callback/") {
			c.Next()
			return
		}

		// Exempt /oauth/token (client credentials flow, no cookies)
		if c.Request.URL.Path == "/oauth/token" || c.Request.URL.Path == "/oauth/revoke" {
			c.Next()
			return
		}

		// Require Content-Type to be application/json or the presence of
		// X-Requested-With header. Browsers cannot send these cross-origin
		// without a CORS preflight check succeeding first.
		contentType := c.Request.Header.Get("Content-Type")
		xRequestedWith := c.Request.Header.Get("X-Requested-With")

		if strings.Contains(contentType, "application/json") || xRequestedWith != "" {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":             "csrf_validation_failed",
			"error_description": "State-changing requests must include Content-Type: application/json or X-Requested-With header",
		})
		c.Abort()
	}
}
