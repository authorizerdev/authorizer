package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/metrics"
)

// ClientCheckMiddleware is a middleware to verify the client ID
// Note: client ID is passed in the header.
// An empty client ID is intentionally allowed for routes that don't require it
// (e.g., OAuth callbacks, JWKS, OpenID configuration, health checks).
// The middleware only rejects requests with an explicitly wrong client ID.
func (h *httpProvider) ClientCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := h.Log.With().Str("func", "ClientCheckMiddleware").
			Str("path", c.Request.URL.Path).
			Logger()
		clientID := c.Request.Header.Get("X-Authorizer-Client-ID")
		if clientID == "" {
			log.Info().Msg("request received without client ID header")
			metrics.RecordClientIDNotFound()
			c.Next()
			return
		}

		if clientID != h.Config.ClientID {
			// Record metric for client-id mismatch, but skip dashboard, admin, and app UI routes
			// as those are internal requests that should not trigger security alerts.
			path := c.Request.URL.Path
			if !strings.HasPrefix(path, "/dashboard") && !strings.HasPrefix(path, "/app") {
				metrics.RecordSecurityEvent("client_id_mismatch", "invalid_client_id")
			}
			log.Debug().Str("client_id", clientID).Msg("Client ID is invalid")
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		c.Next()
	}
}
