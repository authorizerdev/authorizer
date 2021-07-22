package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/yauthdev/yauth/server/constants"
)

func SetCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true

	if !constants.IS_PROD {
		secure = false
	}
	host := GetFrontendHost()

	gc.SetCookie(constants.COOKIE_NAME, token, 3600, "/", host, secure, httpOnly)
}

func DeleteCookie(gc *gin.Context) {
	secure := true
	httpOnly := true

	if !constants.IS_PROD {
		secure = false
	}

	gc.SetCookie(constants.COOKIE_NAME, "", -1, "/", GetFrontendHost(), secure, httpOnly)
}
