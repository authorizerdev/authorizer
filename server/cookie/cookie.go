package cookie

import (
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/gin-gonic/gin"
)

// SetSession sets the session cookie in the response
func SetSession(gc *gin.Context, sessionID string) {
	appCookieSecure, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyAppCookieSecure)
	if err != nil {
		log.Debug("Error while getting app cookie secure from env variable: %v", err)
		appCookieSecure = true
	}

	secure := appCookieSecure
	httpOnly := appCookieSecure
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	// Since app cookie can come from cross site it becomes important to set this in lax mode when insecure.
	// Example person using custom UI on their app domain and making request to authorizer domain.
	// For more information check:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite
	// https://github.com/gin-gonic/gin/blob/master/context.go#L86
	// TODO add ability to sameSite = none / strict from dashboard
	if !appCookieSecure {
		gc.SetSameSite(http.SameSiteLaxMode)
	} else {
		gc.SetSameSite(http.SameSiteNoneMode)
	}
	// TODO allow configuring from dashboard
	year := 60 * 60 * 24 * 365

	gc.SetCookie(constants.AppCookieName+"_session", sessionID, year, "/", host, secure, httpOnly)
	gc.SetCookie(constants.AppCookieName+"_session_domain", sessionID, year, "/", domain, secure, httpOnly)
}

// DeleteSession sets session cookies to expire
func DeleteSession(gc *gin.Context) {
	appCookieSecure, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyAppCookieSecure)
	if err != nil {
		log.Debug("Error while getting app cookie secure from env variable: %v", err)
		appCookieSecure = true
	}

	secure := appCookieSecure
	httpOnly := appCookieSecure
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.AppCookieName+"_session", "", -1, "/", host, secure, httpOnly)
	gc.SetCookie(constants.AppCookieName+"_session_domain", "", -1, "/", domain, secure, httpOnly)
}

// GetSession gets the session cookie from context
func GetSession(gc *gin.Context) (string, error) {
	var cookie *http.Cookie
	var err error
	cookie, err = gc.Request.Cookie(constants.AppCookieName + "_session")
	if err != nil {
		cookie, err = gc.Request.Cookie(constants.AppCookieName + "_session_domain")
		if err != nil {
			return "", err
		}
	}

	decodedValue, err := url.PathUnescape(cookie.Value)
	if err != nil {
		return "", err
	}
	return decodedValue, nil
}
