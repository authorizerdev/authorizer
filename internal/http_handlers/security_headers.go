package http_handlers

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware sets standard security headers on every response.
func (h *httpProvider) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("X-XSS-Protection", "0")
		c.Next()
	}
}
