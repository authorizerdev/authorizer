package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler is the handler for /health route.
// It states if server is in healthy state or not
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	}
}
