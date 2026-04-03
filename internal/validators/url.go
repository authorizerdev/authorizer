package validators

import (
	"net/url"
	"regexp"
	"strings"
)

// normalizeOrigin extracts hostname:port from a URL or origin string.
// Standard ports (80/443) and absent ports are omitted so that
// "https://example.com" and "https://example.com:443" both normalise to "example.com".
func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" || port == "443" || port == "80" {
		return host
	}
	return host + ":" + port
}

// IsValidRedirectURI validates a redirect URI for security-critical flows (password reset,
// magic link, OAuth, etc.). Unlike IsValidOrigin (used for CORS), this function never
// accepts "*" as a blanket pass. When allowed_origins contains only "*" (the default),
// it restricts redirects to the server's own hostname. When explicit origins are
// configured, it validates against those using the same matching logic as IsValidOrigin.
func IsValidRedirectURI(redirectURI string, allowedOrigins []string, hostname string) bool {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	// Only allow http and https schemes to prevent javascript:, data:, ftp:, etc.
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	origins := allowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	redirectOrigin := normalizeOrigin(redirectURI)

	// When allowed_origins is wildcard, only allow redirects to the server's own hostname
	if len(origins) == 1 && origins[0] == "*" {
		return redirectOrigin == normalizeOrigin(hostname)
	}

	// Validate against explicit allowed origins (same logic as IsValidOrigin)
	for _, origin := range origins {
		pattern := normalizeOrigin(origin)

		if strings.Contains(origin, "*") {
			pattern = strings.ReplaceAll(pattern, ".", "\\.")
			pattern = strings.ReplaceAll(pattern, "*", ".*")

			if strings.HasPrefix(pattern, ".*") {
				pattern += "\\b"
			}

			if strings.HasSuffix(pattern, ".*") {
				pattern = "\\b" + pattern
			}
		}

		if matched, _ := regexp.MatchString("^"+pattern+"$", redirectOrigin); matched {
			return true
		}
	}

	return false
}

// IsValidOrigin validates origin based on ALLOWED_ORIGINS
func IsValidOrigin(inputURL string, allowedOriginsConfig []string) bool {
	allowedOrigins := allowedOriginsConfig
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true
	}

	currentOrigin := normalizeOrigin(inputURL)

	for _, origin := range allowedOrigins {
		// Normalize the allowed origin the same way as the input URL
		pattern := normalizeOrigin(origin)

		// if has wildcard domains, convert to regex
		if strings.Contains(origin, "*") {
			pattern = strings.ReplaceAll(pattern, ".", "\\.")
			pattern = strings.ReplaceAll(pattern, "*", ".*")

			if strings.HasPrefix(pattern, ".*") {
				pattern += "\\b"
			}

			if strings.HasSuffix(pattern, ".*") {
				pattern = "\\b" + pattern
			}
		}

		if matched, _ := regexp.MatchString("^"+pattern+"$", currentOrigin); matched {
			return true
		}
	}

	return false
}
