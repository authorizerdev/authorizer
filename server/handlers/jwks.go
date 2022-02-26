package handlers

import (
	"encoding/json"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/gin-gonic/gin"
)

func JWKsHandler() gin.HandlerFunc {
	var data map[string]string
	json.Unmarshal([]byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJWK)), &data)
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"keys": []map[string]string{
				data,
			},
		})
	}
}
