package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// exemptPrefixes are path prefixes that bypass rate limiting.
// These are infrastructure, static asset, and OIDC discovery endpoints.
var exemptPrefixes = []string{
	"/app/",
	"/dashboard/",
}

// exemptPaths are exact paths that bypass rate limiting.
var exemptPaths = map[string]bool{
	"/":                                 true,
	"/health":                           true,
	"/healthz":                          true,
	"/readyz":                           true,
	"/metrics":                          true,
	"/playground":                       true,
	"/.well-known/openid-configuration": true,
	"/.well-known/jwks.json":            true,
}

func isExemptPath(path string) bool {
	if exemptPaths[path] {
		return true
	}
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// RateLimitMiddleware returns a gin middleware that rate limits requests per IP.
func (h *httpProvider) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isExemptPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		if h.Config.RateLimitRPS <= 0 {
			c.Next()
			return
		}

		if h.Dependencies.RateLimitProvider == nil {
			c.Next()
			return
		}

		allowed, err := h.Dependencies.RateLimitProvider.Allow(c.Request.Context(), c.ClientIP())
		if err != nil {
			log := h.Dependencies.Log.With().Str("func", "RateLimitMiddleware").Logger()
			log.Error().Err(err).Msg("rate limit check failed, allowing request")
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", "1")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate_limit_exceeded",
				"error_description": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
