package middlewares

import (
	"context"

	"github.com/gin-gonic/gin"
)

// GinContextToContextMiddleware is a middleware to add gin context in context
func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
