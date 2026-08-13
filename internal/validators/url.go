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

// originMatches reports whether candidate — an already-normalised host[:port] —
// is permitted by one configured allowlist entry.
//
// An exact origin is compared as a STRING. It used to be spliced into a regex:
// `https://login.acme.com` became `^login.acme.com$`, in which every "." matches
// ANY character, so an attacker who registered a lookalike (`loginXacme.com`,
// `login-acme.com`) passed the allowlist and received OAuth access tokens, ID
// tokens and password-reset tokens. Only the wildcard branch escaped its dots;
// the exact branch — the documented production hardening — did not.
//
// A wildcard origin is escaped in full and then has ONLY the "*" re-opened, so
// the wildcard is the single metacharacter that survives from operator config.
//
// Both callers route through here on purpose. They previously carried
// copy-pasted matching blocks, which is how one of them came to escape dots
// while the other did not.
func originMatches(configured, candidate string) bool {
	pattern := normalizeOrigin(configured)
	if !strings.Contains(pattern, "*") {
		return pattern == candidate
	}

	quoted := regexp.QuoteMeta(pattern)
	var expr string
	if rest, ok := strings.CutPrefix(quoted, `\*\.`); ok {
		// Subdomain wildcard: *.example.com must only match proper subdomains
		// (sub.example.com), not evil-example.com and not the bare domain.
		expr = `([^.]+\.)+` + rest
	} else {
		expr = strings.ReplaceAll(quoted, `\*`, `[^.]*`)
	}
	matched, err := regexp.MatchString("^"+expr+"$", candidate)
	return err == nil && matched
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
		if originMatches(origin, redirectOrigin) {
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
		if originMatches(origin, currentOrigin) {
			return true
		}
	}

	return false
}
