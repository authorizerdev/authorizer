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
