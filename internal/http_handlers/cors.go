package http_handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/validators"
)

// CORSMiddleware is a middleware to add cors headers
func (h *httpProvider) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// Under a wildcard allow-list, reflect the literal "*" and send NO
		// credentials header.
		//
		// Reflecting the caller's exact Origin alongside
		// Access-Control-Allow-Credentials: true is functionally "any site may
		// read credentialed responses from this API" — the Fetch spec forbids
		// pairing credentials with a wildcard precisely to stop that, and
		// echoing the origin back is the same thing wearing a disguise. Today
		// the blast radius is small (the authenticated API reads its token from
		// the Authorization header only, GraphQL is POST-behind-CSRF, and the
		// session cookie is HttpOnly), but that is a property of the current
		// endpoints, not of this middleware. The first GET endpoint to adopt
		// cookie auth would turn it into a live cross-origin read.
		//
		// Credentialed CORS therefore requires an explicit allow-list.
		if isWildcardOrigins(h.Config.AllowedOrigins) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && validators.IsValidOrigin(origin, h.Config.AllowedOrigins) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			// The response varies by request Origin, so it must not be cached
			// and replayed to a different origin.
			c.Writer.Header().Add("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With,  X-authorizer-url, X-Forwarded-Proto, X-authorizer-client-id")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
