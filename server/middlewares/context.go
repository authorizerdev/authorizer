package middlewares

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

// GinContextToContextMiddleware is a middleware to add gin context in context
func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAuthorizerURL) == "" {
			url := location.Get(c)
			log.Println("=> setting authorizer url to: " + url.Scheme + "://" + c.Request.Host)
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyAuthorizerURL, url.Scheme+"://"+c.Request.Host)
		}
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
