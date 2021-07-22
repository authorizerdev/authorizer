package utils

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/yauthdev/yauth/server/constants"
)

func SetCookie(gc *gin.Context, token string) {
	secure := true
	httpOnly := true

	if !constants.IS_PROD {
		secure = false
	}
	u, err := url.Parse(constants.SERVER_URL)
	if err != nil {
		log.Println("error getting server host")
	}
	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.COOKIE_NAME, token, 3600, "/", u.Hostname(), secure, httpOnly)
}

func DeleteCookie(gc *gin.Context) {
	secure := true
	httpOnly := true

	if !constants.IS_PROD {
		secure = false
	}

	u, err := url.Parse(constants.SERVER_URL)
	if err != nil {
		log.Println("error getting server host")
	}
	gc.SetSameSite(http.SameSiteNoneMode)

	gc.SetCookie(constants.COOKIE_NAME, "", -1, "/", u.Hostname(), secure, httpOnly)
}
