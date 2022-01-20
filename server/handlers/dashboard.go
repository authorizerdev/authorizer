package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/gin-gonic/gin"
)

// DashboardHandler is the handler for the /dashboard route
func DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		isOnboardingCompleted := false

		if envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret) != "" {
			isOnboardingCompleted = true
		}

		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": isOnboardingCompleted,
			},
		})
	}
}
