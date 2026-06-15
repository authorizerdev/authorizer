package cookie

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// SetAdminCookie sets the admin cookie in the response
func SetAdminCookie(gc *gin.Context, token string, adminCookieSecure bool) {
	c := BuildAdminCookie(parsers.GetHost(gc), token, adminCookieSecure)
	gc.SetSameSite(c.SameSite)
	gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
}

// BuildAdminCookie returns the admin session cookie to set on the response.
// Transport-agnostic so non-gin callers (the service layer, gRPC handlers) can
// produce it as a side-effect. Mirrors SetAdminCookie: host-scoped, HttpOnly,
// SameSite=Strict, 1-hour lifetime.
func BuildAdminCookie(hostname, token string, adminCookieSecure bool) *http.Cookie {
	host, _ := parsers.GetHostParts(hostname)
	return &http.Cookie{
		Name:     constants.AdminCookieName,
		Value:    token,
		MaxAge:   3600,
		Path:     "/",
		Domain:   host,
		Secure:   adminCookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
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
func DeleteAdminCookie(gc *gin.Context, adminCookieSecure bool) {
	c := BuildDeleteAdminCookie(parsers.GetHost(gc), adminCookieSecure)
	gc.SetSameSite(c.SameSite)
	gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
}

// BuildDeleteAdminCookie returns a zero-value, expired admin cookie that causes
// browsers to delete the admin session cookie. Transport-agnostic mirror of
// DeleteAdminCookie.
func BuildDeleteAdminCookie(hostname string, adminCookieSecure bool) *http.Cookie {
	host, _ := parsers.GetHostParts(hostname)
	return &http.Cookie{
		Name:     constants.AdminCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   host,
		Secure:   adminCookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}
