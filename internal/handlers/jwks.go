package handlers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memorystore"
)

func JWKsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		jwk, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJWK)
		if err != nil {
			log.Debug("Error getting JWK from memorystore: ", err)
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		err = json.Unmarshal([]byte(jwk), &data)
		if err != nil {
			log.Debug("Failed to parse JWK: ", err)
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
