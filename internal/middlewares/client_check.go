package middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// ClientCheckMiddleware is a middleware to verify the client ID
// Note: client ID is passed in the header
func ClientCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.Request.Header.Get("X-Authorizer-Client-ID")
		// TODO - uncomment the below code after implementing the client ID check
		fmt.Println("clientID: ", clientID)
		// if client, _ := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID); clientID != "" && client != "" && client != clientID {
		// 	log.Debug("Client ID is invalid: ", clientID)
		// 	c.JSON(http.StatusBadRequest, gin.H{
		// 		"error":             "invalid_client_id",
		// 		"error_description": "The client id is invalid",
		// 	})
		// 	return
		// }

		c.Next()
	}
}
