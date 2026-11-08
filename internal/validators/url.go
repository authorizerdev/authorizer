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

// IsWildcardOrigin reports whether one configured allowlist entry means "any
// host". It is THE single definition of that question: internal/http_handlers'
// CORS and CSRF middleware and cmd's startup warning all call it, because when
// they each had their own `o == "*"` the answers disagreed and the disagreement
// was exploitable.
//
// It decides on the NORMALISED value, not the raw string, because the regex
// splice below also works on the normalised value. "*", " *", "https://*",
// "http://*", "*/", "*/callback", "*:80" and "*:443" are all spellings of the
// same entry: normalizeOrigin strips the scheme, the path, a default port and
// surrounding space, and every one of them collapses to "*". A guard that
// compared the raw string let all but the first walk straight into the regex
// and become [^.]*, which is the whole bug this file exists to fix.
func IsWildcardOrigin(configured string) bool {
	return normalizeOrigin(configured) == "*"
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
	// A bare "*" never belongs here. Both callers special-case it BEFORE this
	// point, with deliberately different meanings — allow-all for CORS,
	// self-host-only for redirects — so there is no single answer this function
	// could return. Letting it through rewrote it as the regex [^.]*, which
	// matched every dotless host (http://3405803783 is 203.0.113.7 to a browser)
	// and no dotted one. Refuse it so a third caller cannot reintroduce that.
	if IsWildcardOrigin(configured) {
		return false
	}

	pattern := normalizeOrigin(configured)
	if !strings.Contains(pattern, "*") {
		return pattern == candidate
	}

	// A wildcard must be anchored by a literal dot somewhere in the entry.
	// Without one the splice below produces a pattern made only of [^.]* and
	// literals that contain no dot — "**" becomes [^.]*[^.]*, "*:3000" becomes
	// [^.]*:3000 — and those match a dotless host, which is the same token-theft
	// primitive as a bare "*": a browser reads http://3405803783 as 203.0.113.7.
	// Every legitimate form keeps its dots ("*.example.com",
	// "*.example.com:8080", "app*.acme.com"), so requiring one costs nothing and
	// closes the whole family at once instead of enumerating spellings.
	if !strings.Contains(pattern, ".") {
		return false
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

	for _, origin := range origins {
		// A bare "*" permits ONLY the server's own hostname, whether it is the
		// whole list or sits beside explicit origins. This is checked per entry,
		// not just for a one-element list: a mixed list used to send "*" into
		// originMatches, where it became [^.]* and accepted any dotless host, so
		// an attacker writing their IPv4 address as a single decimal number
		// received the victim's tokens from /authorize.
		if IsWildcardOrigin(origin) {
			if redirectOrigin == normalizeOrigin(hostname) {
				return true
			}
			continue
		}
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
	currentOrigin := normalizeOrigin(inputURL)

	for _, origin := range allowedOrigins {
		// A bare "*" is allow-all for CORS, whether it is the whole list or sits
		// beside explicit origins. Checked per entry for the same reason as in
		// IsValidRedirectURI: in a mixed list it used to reach originMatches and
		// silently narrow the allowlist to dotless hosts only, which is neither
		// what the operator wrote nor safe.
		if IsWildcardOrigin(origin) {
			return true
		}
		if originMatches(origin, currentOrigin) {
			return true
		}
	}

	return false
}
