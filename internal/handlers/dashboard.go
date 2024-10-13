package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/gin-gonic/gin"
)

// DashboardHandler is the handler for the /dashboard route
func DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		isOnboardingCompleted := false
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		if err != nil || adminSecret != "" {
			isOnboardingCompleted = true
		}

		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": isOnboardingCompleted,
			},
		})
	}
}
