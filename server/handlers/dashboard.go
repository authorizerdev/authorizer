package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
)

func DashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		isOnboardingCompleted := false
		if constants.ADMIN_SECRET != "" && constants.DATABASE_TYPE != "" && constants.DATABASE_URL != "" {
			isOnboardingCompleted = true
		}

		c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{
			"data": map[string]interface{}{
				"isOnboardingCompleted": isOnboardingCompleted,
			},
		})
	}
}
