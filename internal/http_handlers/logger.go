package http_handlers

import (
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware adds logging to Gin using rs/zerolog.
func (h *httpProvider) LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process the request
		c.Next()

		// Log the request and response details
		h.Dependencies.Log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int("size", c.Writer.Size()).
			Str("client_ip", c.ClientIP()).
			Dur("latency", time.Since(start)).
			Msg("HTTP request")
	}
}
