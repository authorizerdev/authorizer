package http_handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ContextMiddleware is a middleware to add gin context in context
func (h *httpProvider) ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
