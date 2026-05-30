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
// Defaults to Lax for unrecognized values. The CLI flag --app-cookie-same-site
// defaults to "none" for backward compatibility with cross-domain SDK setups.
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

// SetSession sets the session cookie in the response.
func SetSession(gc *gin.Context, sessionID string, appCookieSecure bool, sameSite http.SameSite) {
	for _, c := range BuildSessionCookies(parsers.GetHost(gc), sessionID, appCookieSecure, sameSite) {
		gc.SetSameSite(c.SameSite)
		gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
	}
}

// BuildSessionCookies returns the pair of session cookies (host-scoped and
// domain-scoped) to set on the response. Transport-agnostic so non-gin
// callers (the service layer, gRPC handlers) can produce them as side-effects.
func BuildSessionCookies(hostname, sessionID string, appCookieSecure bool, sameSite http.SameSite) []*http.Cookie {
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}
	day := 60 * 60 * 24
	return []*http.Cookie{
		{
			Name:     constants.AppCookieName + "_session",
			Value:    sessionID,
			MaxAge:   day,
			Path:     "/",
			Domain:   host,
			Secure:   appCookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		},
		{
			Name:     constants.AppCookieName + "_session_domain",
			Value:    sessionID,
			MaxAge:   day,
			Path:     "/",
			Domain:   domain,
			Secure:   appCookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		},
	}
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
