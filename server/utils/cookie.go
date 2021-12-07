package utils

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
)

func SetCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true
	origin := constants.APP_URL

	host := GetHostName(constants.AUTHORIZER_URL)
	originHost := GetHostName(origin)

	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.COOKIE_NAME, token, 3600, "/", host, secure, httpOnly)
	gc.SetCookie(constants.COOKIE_NAME+"-client", token, 3600, "/", originHost, secure, httpOnly)
}

func GetCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(constants.COOKIE_NAME)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func DeleteCookie(gc *gin.Context) {
	secure := true
	httpOnly := true
	origin := constants.APP_URL

	if !constants.IS_PROD {
		secure = false
	}

	host := GetHostName(constants.AUTHORIZER_URL)
	originHost := GetHostName(origin)
	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.COOKIE_NAME, "", -1, "/", host, secure, httpOnly)
	gc.SetCookie(constants.COOKIE_NAME+"-client", "", -1, "/", originHost, secure, httpOnly)
}
