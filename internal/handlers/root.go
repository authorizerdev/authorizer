package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RootHandler is the handler for / root route.
func RootHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
	}
}
