package cookie

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// SetMfaSession sets the mfa session cookie in the response. expiresAt is the
// same absolute Unix timestamp passed to MemoryStoreProvider.SetMfaSession -
// the cookie's MaxAge must match the underlying session's actual TTL, or the
// browser deletes the cookie before the session it points to expires (e.g. a
// user who takes over a minute to read an MFA offer screen and click "Skip
// for now" would get a valid session but a browser-deleted cookie).
func SetMfaSession(gc *gin.Context, sessionID string, appCookieSecure bool, expiresAt int64) {
	for _, c := range BuildMfaSessionCookies(parsers.GetHost(gc), sessionID, appCookieSecure, expiresAt) {
		gc.SetSameSite(c.SameSite)
		gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
	}
}

// BuildMfaSessionCookies returns the MFA session cookies (host-scoped and
// domain-scoped) to set on the response. Transport-agnostic mirror of
// SetMfaSession. expiresAt must match the caller's MemoryStoreProvider.
// SetMfaSession call - see that function's doc comment for why.
//
// SameSite policy mirrors the gin path: Lax when insecure (so cross-site UI
// can still complete the flow), None when secure. See the SetMfaSession
// comment for the historical reasoning and the configurability TODO.
func BuildMfaSessionCookies(hostname, sessionID string, appCookieSecure bool, expiresAt int64) []*http.Cookie {
	host, _ := parsers.GetHostParts(hostname)
	domain := parsers.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}
	sameSite := http.SameSiteNoneMode
	if !appCookieSecure {
		sameSite = http.SameSiteLaxMode
	}
	age := int(time.Until(time.Unix(expiresAt, 0)).Seconds())
	return []*http.Cookie{
		{
			Name:     constants.MfaCookieName + "_session",
			Value:    sessionID,
			MaxAge:   age,
			Path:     "/",
			Domain:   host,
			Secure:   appCookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		},
		{
			Name:     constants.MfaCookieName + "_session_domain",
			Value:    sessionID,
			MaxAge:   age,
			Path:     "/",
			Domain:   domain,
			Secure:   appCookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		},
	}
}

// DeleteMfaSession deletes the mfa session cookies to expire
func DeleteMfaSession(gc *gin.Context, appCookieSecure bool) {
	secure := appCookieSecure
	httpOnly := true
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
