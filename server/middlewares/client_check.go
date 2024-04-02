package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// ClientCheckMiddleware is a middleware to verify the client ID
// Note: client ID is passed in the header
func ClientCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.Request.Header.Get("X-Authorizer-Client-ID")
		if client, _ := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID); clientID != "" && client != "" && client != clientID {
			log.Debug("Client ID is invalid: ", clientID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		c.Next()
	}
}
