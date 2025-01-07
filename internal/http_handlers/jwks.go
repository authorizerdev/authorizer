package http_handlers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func (h *httpProvider) JWKsHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "JWKsHandler").Logger()
	return func(c *gin.Context) {
		var data map[string]string
		// TODO
		jwk := h.Config.JWTPublicKey
		err := json.Unmarshal([]byte(jwk), &data)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal jwk")
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"keys": []map[string]string{
				data,
			},
		})
	}
}
