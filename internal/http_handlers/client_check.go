package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClientCheckMiddleware is a middleware to verify the client ID
// Note: client ID is passed in the header.
// An empty client ID is intentionally allowed for routes that don't require it
// (e.g., OAuth callbacks, JWKS, OpenID configuration, health checks).
// The middleware only rejects requests with an explicitly wrong client ID.
func (h *httpProvider) ClientCheckMiddleware() gin.HandlerFunc {
	log := h.Log.With().Str("func", "ClientCheckMiddleware").Logger()
	return func(c *gin.Context) {
		clientID := c.Request.Header.Get("X-Authorizer-Client-ID")
		if clientID == "" {
			log.Debug().Msg("request received without client ID header")
			c.Next()
			return
		}

		if clientID != h.Config.ClientID {
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
