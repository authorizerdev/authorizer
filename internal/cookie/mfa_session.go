package cookie

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// TODO set app cookie as per config

// SetMfaSession sets the mfa session cookie in the response
func SetMfaSession(gc *gin.Context, sessionID string, appCookieSecure bool) {
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
	age := 60

	gc.SetCookie(constants.MfaCookieName+"_session", sessionID, age, "/", host, secure, httpOnly)
	gc.SetCookie(constants.MfaCookieName+"_session_domain", sessionID, age, "/", domain, secure, httpOnly)
}

// DeleteMfaSession deletes the mfa session cookies to expire
func DeleteMfaSession(gc *gin.Context, appCookieSecure bool) {
	secure := appCookieSecure
	httpOnly := appCookieSecure
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(constants.MfaCookieName+"_session", "", -1, "/", host, secure, httpOnly)
	gc.SetCookie(constants.MfaCookieName+"_session_domain", "", -1, "/", domain, secure, httpOnly)
}

// GetMfaSession gets the mfa session cookie from context
func GetMfaSession(gc *gin.Context) (string, error) {
	var cookie *http.Cookie
	var err error
	cookie, err = gc.Request.Cookie(constants.MfaCookieName + "_session")
	if err != nil {
		cookie, err = gc.Request.Cookie(constants.MfaCookieName + "_session_domain")
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
