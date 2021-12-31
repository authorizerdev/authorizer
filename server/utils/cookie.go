package utils

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
)

func SetCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true
	host, _ := GetHostParts(constants.EnvData.AUTHORIZER_URL)
	domain := GetDomainName(constants.EnvData.AUTHORIZER_URL)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.EnvData.COOKIE_NAME, token, 3600, "/", host, secure, httpOnly)
	gc.SetCookie(constants.EnvData.COOKIE_NAME+"-client", token, 3600, "/", domain, secure, httpOnly)
}

func GetCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(constants.EnvData.COOKIE_NAME)
	if err != nil {
		cookie, err = gc.Request.Cookie(constants.EnvData.COOKIE_NAME + "-client")
		if err != nil {
			return "", err
		}
	}

	return cookie.Value, nil
}

func DeleteCookie(gc *gin.Context) {
	secure := true
	httpOnly := true

	host, _ := GetHostParts(constants.EnvData.AUTHORIZER_URL)
	domain := GetDomainName(constants.EnvData.AUTHORIZER_URL)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.EnvData.COOKIE_NAME, "", -1, "/", host, secure, httpOnly)
	gc.SetCookie(constants.EnvData.COOKIE_NAME+"-client", "", -1, "/", domain, secure, httpOnly)
}

func SetAdminCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true
	host, _ := GetHostParts(constants.EnvData.AUTHORIZER_URL)

	gc.SetCookie(constants.EnvData.ADMIN_COOKIE_NAME, token, 3600, "/", host, secure, httpOnly)
}

func GetAdminCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(constants.EnvData.ADMIN_COOKIE_NAME)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func DeleteAdminCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true
	host, _ := GetHostParts(constants.EnvData.AUTHORIZER_URL)

	gc.SetCookie(constants.EnvData.ADMIN_COOKIE_NAME, "", -1, "/", host, secure, httpOnly)
}
