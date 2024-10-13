package cookie

import (
	"net/url"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/gin-gonic/gin"
)

// SetAdminCookie sets the admin cookie in the response
func SetAdminCookie(gc *gin.Context, token string) {
	adminCookieSecure, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyAdminCookieSecure)
	if err != nil {
		log.Debug("Error while getting admin cookie secure from env variable: %v", err)
		adminCookieSecure = true
	}

	secure := adminCookieSecure
	httpOnly := adminCookieSecure
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	gc.SetCookie(constants.AdminCookieName, token, 3600, "/", host, secure, httpOnly)
}

// GetAdminCookie gets the admin cookie from the request
func GetAdminCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(constants.AdminCookieName)
	if err != nil {
		return "", err
	}

	// cookie escapes special characters like $
	// hence we need to unescape before comparing
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return "", err
	}
	return decodedValue, nil
}

// DeleteAdminCookie sets the response cookie to empty
func DeleteAdminCookie(gc *gin.Context) {
	adminCookieSecure, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyAdminCookieSecure)
	if err != nil {
		log.Debug("Error while getting admin cookie secure from env variable: %v", err)
		adminCookieSecure = true
	}

	secure := adminCookieSecure
	httpOnly := adminCookieSecure
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	gc.SetCookie(constants.AdminCookieName, "", -1, "/", host, secure, httpOnly)
}
