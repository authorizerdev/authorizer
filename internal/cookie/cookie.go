package cookie

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// ParseSameSite converts a string ("lax", "strict", "none") to http.SameSite.
// Defaults to Lax for unrecognized values.
func ParseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}

// SetSession sets the session cookie in the response
func SetSession(gc *gin.Context, sessionID string, appCookieSecure bool, sameSite http.SameSite) {
	secure := appCookieSecure
	httpOnly := true
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(sameSite)
	day := 60 * 60 * 24

	gc.SetCookie(constants.AppCookieName+"_session", sessionID, day, "/", host, secure, httpOnly)
	gc.SetCookie(constants.AppCookieName+"_session_domain", sessionID, day, "/", domain, secure, httpOnly)
}

// DeleteSession sets session cookies to expire
func DeleteSession(gc *gin.Context, appCookieSecure bool, sameSite http.SameSite) {
	secure := appCookieSecure
	httpOnly := true
	hostname := parsers.GetHost(gc)
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(sameSite)
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
