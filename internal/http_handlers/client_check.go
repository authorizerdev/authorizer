package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClientCheckMiddleware is a middleware to verify the client ID
// Note: client ID is passed in the header
func (h *httpProvider) ClientCheckMiddleware() gin.HandlerFunc {
	log := h.Log.With().Str("func", "ClientCheckMiddleware").Logger()
	return func(c *gin.Context) {
		clientID := c.Request.Header.Get("X-Authorizer-Client-ID")
		if clientID != "" && clientID != h.Config.ClientID {
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
