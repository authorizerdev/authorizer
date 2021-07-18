package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/yauthdev/yauth/server/constants"
)

func IsSuperAdmin(gc *gin.Context) bool {
	secret := gc.Request.Header.Get("x-yauth-admin-secret")
	if secret == "" {
		return false
	}

	return secret == constants.YAUTH_ADMIN_SECRET
}
