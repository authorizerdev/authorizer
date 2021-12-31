package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
)

func DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		isOnboardingCompleted := false
		if constants.EnvData.ADMIN_SECRET != "" && constants.EnvData.DATABASE_TYPE != "" && constants.EnvData.DATABASE_URL != "" {
			isOnboardingCompleted = true
		}

		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": isOnboardingCompleted,
			},
		})
	}
}
