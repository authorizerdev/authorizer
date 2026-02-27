package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DashboardHandler is the handler for the /dashboard route
func (h *httpProvider) DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": true,
			},
		})
	}
}
