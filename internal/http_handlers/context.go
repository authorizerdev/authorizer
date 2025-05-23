package http_handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Define a custom type for context key
type contextKey string

const ginContextKey contextKey = "GinContextKey"

// ContextMiddleware is a middleware to add gin context in context
func (h *httpProvider) ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ginContextKey, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
