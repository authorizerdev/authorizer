package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
)

func IsSuperAdmin(gc *gin.Context) bool {
	secret := gc.Request.Header.Get("x-authorizer-admin-secret")
	if secret == "" {
		return false
	}

	return secret == constants.ADMIN_SECRET
}
