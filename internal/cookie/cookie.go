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
//
// # Why there are two cookies, and why SameSite defaults to None
//
// DELIBERATE, and reviewed. The 2.4.0 pre-release security audit flagged both
// as hardening opportunities — default SameSite=Lax, drop the domain-scoped
// twin — and both were consciously declined. Please do not "fix" them without
// re-reading this.
//
// Authorizer's intended deployment is an auth server on a subdomain
// (auth.example.com) serving several apps, some on sibling subdomains and some
// on entirely different domains. Both properties exist to serve that:
//
//   - the domain-scoped (".example.com") twin is what lets app.example.com see
//     a session established at auth.example.com. Dropping it breaks subdomain
//     SSO outright.
//   - SameSite=None is what lets an app on a DIFFERENT site complete a
//     credentialed /session call at all. Lax withholds the cookie on exactly
//     those cross-site requests, so the browser-session half of the SDK stops
//     working.
//
// Auth0 lands in the same place: it recommends SameSite=None for cross-origin
// authentication (it is required when response_mode=form_post), and ships
// fallback cookies — auth0_compat and friends — for browsers that cannot do
// SameSite=None at all. Its documented answer to third-party-cookie blocking is
// Custom Domains: put the auth server on the customer's own subdomain so its
// cookies are first-party. That is precisely the topology above, and precisely
// what the domain-scoped cookie here provides.
//
// The security cost is understood and covered elsewhere: CSRF middleware
// (internal/http_handlers/csrf.go) is the primary defense for state-changing
// requests, the cookie is HttpOnly, the authenticated API reads its token from
// the Authorization header rather than this cookie, and the admin cookie is
// independently SameSite=Strict. SameSite here is defense-in-depth, not the
// control being relied upon.
//
// Operators who do NOT need cross-site apps should set
// --app-cookie-same-site=lax. That is the knob; the default is chosen for the
// topology the product targets.
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

// DeleteSession sets session cookies to expire.
func DeleteSession(gc *gin.Context, appCookieSecure bool, sameSite http.SameSite) {
	for _, c := range BuildDeleteSessionCookies(parsers.GetHost(gc), appCookieSecure, sameSite) {
		gc.SetSameSite(c.SameSite)
		gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
	}
}

// BuildDeleteSessionCookies returns the pair of zero-value, expired session
// cookies that cause browsers to delete the host-scoped and domain-scoped
// session cookies. Transport-agnostic mirror of DeleteSession.
func BuildDeleteSessionCookies(hostname string, appCookieSecure bool, sameSite http.SameSite) []*http.Cookie {
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}
	return []*http.Cookie{
		{
			Name:     constants.AppCookieName + "_session",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			Domain:   host,
			Secure:   appCookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		},
		{
			Name:     constants.AppCookieName + "_session_domain",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			Domain:   domain,
			Secure:   appCookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		},
	}
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
