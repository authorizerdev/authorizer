package http_handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/validators"
)

// CORSMiddleware is a middleware to add cors headers
func (h *httpProvider) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// TODO set valid origins as per config
		if validators.IsValidOrigin(origin, h.Config.AllowedOrigins) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With,  X-authorizer-url, X-Forwarded-Proto, X-authorizer-client-id")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
