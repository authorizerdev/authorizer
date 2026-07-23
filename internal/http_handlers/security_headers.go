package http_handlers

import (
	"github.com/gin-gonic/gin"
)

// defaultCSP is a conservative starting policy. 'unsafe-inline' is required
// today because the dashboard uses inline styles; tighten as the frontend
// migrates away from inline. frame-ancestors 'none' replaces X-Frame-Options.
const defaultCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' https://editor.unlayer.com; " +
	"style-src 'self' 'unsafe-inline' https://editor.unlayer.com; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' data: https://editor.unlayer.com; " +
	"connect-src 'self' https://api.unlayer.com; " +
	"frame-src https://editor.unlayer.com; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

// playgroundCSP relaxes script/style/font sources for the embedded GraphQL
// playground, which loads React + GraphiQL bundles from jsdelivr at runtime.
// Scoped to /playground only — the rest of the app keeps defaultCSP.
const playgroundCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
	"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' data: https://cdn.jsdelivr.net; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

// samlIDPSSOCSP omits form-action for the SAML IdP SSO endpoints. Their
// response (crewjam WriteResponse) is a self-submitting HTML form that must
// POST to the requesting SP's ACS URL — an arbitrary external origin by
// design, so form-action 'self' would silently break every SAML IdP login.
// This is safe to relax: form-action does not inherit from default-src, so
// omitting it lifts the restriction only for this response, and the ACS
// destination is never request-supplied — it's the SP's own URL, validated
// server-side against the org's registered SP record before the response is
// built (see saml.IdpAuthnRequest.Validate and
// GetSAMLServiceProviderByOrgAndEntityID in saml_idp.go).
const samlIDPSSOCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'"

// SecurityHeadersMiddleware sets standard security headers on every response.
func (h *httpProvider) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hdr := c.Writer.Header()
		hdr.Set("X-Content-Type-Options", "nosniff")
		hdr.Set("X-Frame-Options", "DENY")
		hdr.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		hdr.Set("X-XSS-Protection", "0")
		hdr.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")

		// HSTS is opt-in via config because operators not behind TLS would
		// lock browsers out for a year. Only emit when explicitly enabled.
		if h.Config.EnableHSTS {
			hdr.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// CSP is on by default; disable via --disable-csp if it breaks a
		// dashboard in the wild while we tighten the policy.
		if !h.Config.DisableCSP {
			csp := defaultCSP
			switch {
			case c.Request.URL.Path == "/playground":
				csp = playgroundCSP
			case c.FullPath() == "/saml/idp/:org_slug/sso" || c.FullPath() == "/saml/idp/:org_slug/sso/:sp_id":
				csp = samlIDPSSOCSP
			}
			hdr.Set("Content-Security-Policy", csp)
		}

		c.Next()
	}
}
