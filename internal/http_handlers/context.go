package http_handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/utils"
)

// ContextMiddleware is a middleware to add gin context in context
func (h *httpProvider) ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := utils.ContextWithGin(c.Request.Context(), c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
