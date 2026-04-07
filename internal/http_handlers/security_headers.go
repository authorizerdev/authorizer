package http_handlers

import (
	"github.com/gin-gonic/gin"
)

// defaultCSP is a conservative starting policy. 'unsafe-inline' is required
// today because the dashboard uses inline styles; tighten as the frontend
// migrates away from inline. frame-ancestors 'none' replaces X-Frame-Options.
const defaultCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

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
			hdr.Set("Content-Security-Policy", defaultCSP)
		}

		c.Next()
	}
}
