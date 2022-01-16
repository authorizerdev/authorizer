package middlewares

import (
	"context"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

// GinContextToContextMiddleware is a middleware to add gin context in context
func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) == "" {
			url := location.Get(c)
			envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.EnvKeyAuthorizerURL, url.Scheme+"://"+c.Request.Host)
		}
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
