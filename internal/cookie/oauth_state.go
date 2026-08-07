package cookie

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// oauthStateCookieMaxAge bounds how long a half-finished social login stays
// resumable. Long enough for a slow consent screen, short enough that a stale
// binding does not linger.
const oauthStateCookieMaxAge = 15 * 60

// SetOAuthState binds an in-flight social login to the browser that started it.
//
// The `state` parameter alone does not do this: it is generated server-side and
// stored globally, so the callback could only ever check that SOME flow issued
// it, not that THIS browser did. That gap is login CSRF (RFC 9700 §4.7) — an
// attacker starts a flow, harvests their own valid code+state, and delivers it
// to a victim's browser, silently logging the victim into the ATTACKER's
// account. Anything the victim then does (saved addresses, payment details,
// uploaded documents) lands in an account the attacker controls.
//
// SameSite is None when the cookie is Secure, matching the MFA session cookie:
// Apple returns its callback as a cross-site form_post (the router accepts POST
// on /oauth_callback for exactly this), and a Lax cookie is not sent on a
// cross-site POST — Apple logins would break. The binding property does not
// depend on SameSite: an attacker cannot write a cookie into the victim's
// browser for our origin, whatever the SameSite value.
func SetOAuthState(gc *gin.Context, state string, appCookieSecure bool) {
	c := BuildOAuthStateCookie("", state, appCookieSecure)
	gc.SetSameSite(c.SameSite)
	gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
}

// BuildOAuthStateCookie returns the state-binding cookie. Host-scoped
// deliberately: the callback runs on this exact host, so there is no reason to
// widen the cookie to sibling subdomains.
// It deliberately does NOT take the operator's --app-cookie-same-site setting.
// SameSite=Strict would withhold this cookie on the provider's callback, which
// arrives as a cross-site redirect (or, for Apple, a cross-site form_post) —
// every social login would fail with "invalid oauth state" on any deployment
// configured strict. Lax/None is a correctness requirement here, not a
// preference, so wiring the session-cookie setting through would be a silent
// outage. TestOAuthStateCookieIsNeverStrict guards that.
func BuildOAuthStateCookie(_ string, state string, appCookieSecure bool) *http.Cookie {
	sameSite := http.SameSiteLaxMode
	if appCookieSecure {
		sameSite = http.SameSiteNoneMode
	}
	return &http.Cookie{
		Name:  constants.OAuthStateCookieName,
		Value: state,
		// Deliberately no Domain: a host-only cookie. The callback runs on the
		// exact host that set this, so widening to sibling subdomains buys
		// nothing and leaks the binding to them. It also sidesteps the cases
		// where a Domain attribute is silently dropped and the cookie never
		// comes back at all — single-label hosts (`authorizer` in the compose
		// network) and bare IPs both hit that.
		MaxAge:   oauthStateCookieMaxAge,
		Path:     "/",
		Secure:   appCookieSecure,
		HttpOnly: true,
		SameSite: sameSite,
	}
}

// GetOAuthState reads the state-binding cookie off the request.
//
// Unescaped on the way out because gin's SetCookie URL-escapes on the way in
// and nothing reverses it on read — the state contains "://" and spaces, so a
// raw comparison against the state parameter never matches. GetAdminCookie
// handles the same asymmetry for the same reason.
func GetOAuthState(gc *gin.Context) string {
	c, err := gc.Request.Cookie(constants.OAuthStateCookieName)
	if err != nil {
		return ""
	}
	decoded, err := url.QueryUnescape(c.Value)
	if err != nil {
		return ""
	}
	return decoded
}

// DeleteOAuthState clears the binding once the flow completes, so a single
// browser cannot replay a spent state.
func DeleteOAuthState(gc *gin.Context, appCookieSecure bool) {
	c := BuildOAuthStateCookie("", "", appCookieSecure)
	c.MaxAge = -1
	gc.SetSameSite(c.SameSite)
	gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
}
