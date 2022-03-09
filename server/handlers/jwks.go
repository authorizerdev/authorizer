package handlers

import (
	"encoding/json"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/gin-gonic/gin"
)

func JWKsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var data map[string]string
		jwk := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJWK)
		err := json.Unmarshal([]byte(jwk), &data)
		if err != nil {
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
