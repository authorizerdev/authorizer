package handlers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
)

func JWKsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		jwk := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJWK)
		err := json.Unmarshal([]byte(jwk), &data)
		if err != nil {
			log.Debug("Failed to parse JWK", err)
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
