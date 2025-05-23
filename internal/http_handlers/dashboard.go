package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DashboardHandler is the handler for the /dashboard route
func (h *httpProvider) DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// SignUp of admin is deprecated
		// isOnboardingCompleted := false
		// adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		// if !s.Config != "" {
		// 	isOnboardingCompleted = true
		// }

		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": true,
			},
		})
	}
}
