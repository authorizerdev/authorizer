package cookie

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// authorizeBindingCookieMaxAge outlives the authorize state it binds (10
// minutes in the memory store) and the magic link that completes it is useless
// once that state has expired, so nothing is gained by keeping it longer.
const authorizeBindingCookieMaxAge = 15 * 60

// SetAuthorizeBinding binds an /authorize request that detours through the
// login UI to the browser that made it.
//
// Most ways back from that detour authenticate the browser itself (a password
// login, an existing session, a social callback checked against
// SetOAuthState). A magic link does not: magic_link_login is unauthenticated
// and accepts any `state`, and the link is clicked wherever the mailbox owner
// opens it. /verify_email therefore issues an authorization code only when the
// clicking browser presents this cookie and its digest matches the one stored
// in the authorize state (codestate.Authorize.BrowserBinding).
//
// SameSite=Lax regardless of --app-cookie-same-site: the click arrives as a
// top-level cross-site GET navigation from a mail client or webmail, which Lax
// allows and Strict would withhold. Host-only for the same reasons as the OAuth
// state cookie: /authorize and /verify_email run on this exact host.
func SetAuthorizeBinding(gc *gin.Context, value string, appCookieSecure bool) {
	c := BuildAuthorizeBindingCookie(value, appCookieSecure)
	gc.SetSameSite(c.SameSite)
	gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
}

// BuildAuthorizeBindingCookie returns the authorize-binding cookie.
func BuildAuthorizeBindingCookie(value string, appCookieSecure bool) *http.Cookie {
	return &http.Cookie{
		Name:     constants.AuthorizeBindingCookieName,
		Value:    value,
		MaxAge:   authorizeBindingCookieMaxAge,
		Path:     "/",
		Secure:   appCookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// GetAuthorizeBinding reads the authorize-binding cookie off the request, or
// "" when absent. The value is URL-safe, so no unescaping is needed.
func GetAuthorizeBinding(gc *gin.Context) string {
	c, err := gc.Request.Cookie(constants.AuthorizeBindingCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// DeleteAuthorizeBinding clears the binding once a code has been issued.
func DeleteAuthorizeBinding(gc *gin.Context, appCookieSecure bool) {
	c := BuildAuthorizeBindingCookie("", appCookieSecure)
	c.MaxAge = -1
	gc.SetSameSite(c.SameSite)
	gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
}
