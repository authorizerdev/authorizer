package utils

import (
	"log"
	"net/http"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
)

func SetCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true

	host := GetDomainName(gc.Request.Host)
	log.Println("=> host", host)
	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.COOKIE_NAME, token, 3600, "/", host, secure, httpOnly)
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

	if !constants.IS_PROD {
		secure = false
	}

	host := GetDomainName(gc.Request.Host)
	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.COOKIE_NAME, "", -1, "/", host, secure, httpOnly)
}
