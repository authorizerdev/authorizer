package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DashboardHandler is the handler for the /dashboard route.
//
// The shell HTML is the entry point that references the (content-hashed)
// SPA bundle. We send Cache-Control: no-cache so browsers always revalidate
// — otherwise after a deploy users may hold a cached shell that points at
// chunks the new build no longer publishes, breaking the app.
func (h *httpProvider) DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, must-revalidate")
		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": true,
			},
		})
	}
}
