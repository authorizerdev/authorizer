package http_handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/validators"
)

// CSRFMiddleware protects against CSRF on state-changing requests
// (POST, PUT, PATCH, DELETE).
//
// Two layers of defence are required because either alone is bypassable:
//
//  1. Origin / Referer must be present and match a non-wildcard allowlist
//     entry, OR (when AllowedOrigins == ["*"]) match the request Host.
//     A wildcard CORS policy must NOT translate to "skip CSRF checks" — the
//     attacker controls the Origin header from a malicious page, so we
//     enforce same-origin in wildcard mode.
//
//  2. Either Content-Type: application/json or X-Requested-With must be
//     present. These are non-CORS-safelisted, so a browser cannot send
//     them cross-origin without a successful preflight, which the Origin
//     check above already covers.
//
// OAuth callback POST routes and the token endpoint are exempt because they
// either originate from provider redirects or use bearer credentials, not
// cookies, so CSRF is not applicable.
func (h *httpProvider) CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Exempt OAuth callback routes (provider POST redirects)
		if strings.HasPrefix(c.Request.URL.Path, "/oauth_callback/") {
			c.Next()
			return
		}

		// Exempt /oauth/token and /oauth/revoke (client credentials flow,
		// authenticated via bearer or client_secret, not cookies)
		if c.Request.URL.Path == "/oauth/token" || c.Request.URL.Path == "/oauth/revoke" {
			c.Next()
			return
		}

		// === Origin / Referer enforcement ===
		// Browsers always send Origin on cross-origin POST. A missing
		// Origin header on a state-changing request is suspicious and
		// indicates either a same-origin tool or an attacker; require at
		// least Referer as a fallback.
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			// Try to derive an origin from the Referer header
			if ref := c.Request.Header.Get("Referer"); ref != "" {
				if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
					origin = u.Scheme + "://" + u.Host
				}
			}
		}
		if origin == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":             "csrf_validation_failed",
				"error_description": "Origin or Referer header is required for state-changing requests",
			})
			c.Abort()
			return
		}

		if !csrfOriginAllowed(origin, c.Request, h.Config.AllowedOrigins) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":             "csrf_validation_failed",
				"error_description": "Origin not allowed",
			})
			c.Abort()
			return
		}

		// === Custom-header enforcement ===
		// Require Content-Type: application/json or X-Requested-With.
		// Browsers cannot set these cross-origin without preflight.
		contentType := c.Request.Header.Get("Content-Type")
		xRequestedWith := c.Request.Header.Get("X-Requested-With")
		if !(strings.Contains(contentType, "application/json") || xRequestedWith != "") {
			c.JSON(http.StatusForbidden, gin.H{
				"error":             "csrf_validation_failed",
				"error_description": "State-changing requests must include Content-Type: application/json or X-Requested-With header",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// csrfOriginAllowed enforces a stricter rule than CORS: when AllowedOrigins
// is the wildcard ["*"], state-changing requests must come from the same
// origin as the server (matching Host). Wildcard CORS does NOT mean wildcard
// CSRF — the attacker controls the Origin header from a malicious page.
func csrfOriginAllowed(origin string, r *http.Request, allowedOrigins []string) bool {
	// Wildcard mode: require same-origin
	if isWildcardOrigins(allowedOrigins) {
		// Compare scheme://host[:port] of origin to the request's own host
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		// Trust the Host header for same-origin comparison. SetTrustedProxies
		// (handled separately) ensures Host has not been spoofed via XFF.
		return strings.EqualFold(u.Host, r.Host)
	}
	// Explicit allowlist: defer to existing matcher
	return validators.IsValidOrigin(origin, allowedOrigins)
}

func isWildcardOrigins(allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, o := range allowed {
		if o == "*" {
			return true
		}
	}
	return false
}
