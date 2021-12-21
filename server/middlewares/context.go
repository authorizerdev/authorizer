package middlewares

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if constants.AUTHORIZER_URL == "" {
			url := location.Get(c)
			constants.AUTHORIZER_URL = url.Scheme + "://" + c.Request.Host
			log.Println("=> authorizer url:", constants.AUTHORIZER_URL)
		}
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
