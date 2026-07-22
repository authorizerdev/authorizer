package parsers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetHost returns the authorizer host URL from the gin request context.
// Thin shim over GetHostFromRequest so non-gin transports (gRPC, plain HTTP)
// can reuse the same host-derivation logic.
func GetHost(c *gin.Context) string {
	return GetHostFromRequest(c.Request)
}

// trustedURL, when non-empty, is the operator-configured canonical base URL
// (--url). It is set ONCE at startup via SetTrustedURL, before any listener
// accepts a connection, so no lock is needed: the write happens-before every
// concurrent read. When set, ALL request headers are ignored for host
// derivation, closing the host-header-injection account-takeover class
// (CWE-640). When empty (default) the legacy header-based derivation below is
// used, preserving reverse-proxy / multi-tenant deployments — operators SHOULD
// set --url; omitting it is what leaves this attack surface open.
// ponytail: set-once-at-startup global; a mutex/atomic would only matter if we
// ever reconfigured this at runtime, which we don't.
var trustedURL string

// SetTrustedURL records the operator-configured canonical base URL. Called once
// from cmd/root at startup. The value is normalized to scheme+host (path,
// query, fragment, userinfo and trailing slash stripped) so it is a valid,
// consistent issuer; an unparsable/invalid value is treated as unset, keeping
// the legacy header-based behavior rather than emitting a broken host.
func SetTrustedURL(u string) {
	trustedURL = sanitizeAuthorizerURL(strings.TrimSpace(u))
}

// GetHostFromRequest returns the authorizer host URL from a raw *http.Request.
// When a trusted URL is configured (--url), it wins over every request header.
// Otherwise: X-Authorizer-URL header, then scheme (X-Forwarded-Proto) + host
// (X-Forwarded-Host or Request.Host). Headers are validated to prevent host
// header injection attacks.
func GetHostFromRequest(r *http.Request) string {
	if trustedURL != "" {
		return trustedURL
	}
	authorizerURL := strings.TrimSpace(r.Header.Get("X-Authorizer-URL"))
	if authorizerURL != "" {
		if sanitized := sanitizeAuthorizerURL(authorizerURL); sanitized != "" {
			return sanitized
		}
		// Invalid header value — fall through to standard host detection
	}

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme != "https" {
		scheme = "http"
	}
	host := sanitizeHost(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = sanitizeHost(r.Host)
	}
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host
}

// sanitizeAuthorizerURL validates and sanitizes the X-Authorizer-URL header.
// Returns empty string if the URL is invalid or contains suspicious components.
func sanitizeAuthorizerURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	// Only allow http/https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	// Must have a valid host
	if u.Host == "" {
		return ""
	}
	// Reject URLs with user info (user:pass@host)
	if u.User != nil {
		return ""
	}
	// Reconstruct with only scheme + host (strip path, query, fragment)
	return strings.TrimSuffix(u.Scheme+"://"+u.Host, "/")
}

// sanitizeHost validates a host header value, rejecting values with
// path components, query strings, or other injection attempts.
func sanitizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	// Reject if it contains path separators, query, or fragment
	if strings.ContainsAny(host, "/?#@\\") {
		return ""
	}
	// Reject if it contains newlines or carriage returns (header injection)
	if strings.ContainsAny(host, "\r\n") {
		return ""
	}
	return strings.TrimSuffix(host, "/")
}

// GetHostParts returns the hostname and port for the given URI.
func GetHostParts(uri string) (string, string) {
	tempURI := uri
	if !strings.HasPrefix(tempURI, "http://") && !strings.HasPrefix(tempURI, "https://") {
		tempURI = "https://" + tempURI
	}

	u, err := url.Parse(tempURI)
	if err != nil {
		return "localhost", "8080"
	}

	host := u.Hostname()
	port := u.Port()

	return host, port
}

// GetDomainName function to get domain name
func GetDomainName(uri string) string {
	tempURI := uri
	if !strings.HasPrefix(tempURI, "http://") && !strings.HasPrefix(tempURI, "https://") {
		tempURI = "https://" + tempURI
	}

	u, err := url.Parse(tempURI)
	if err != nil {
		return `localhost`
	}

	host := u.Hostname()

	// code to get root domain in case of sub-domains
	hostParts := strings.Split(host, ".")
	hostPartsLen := len(hostParts)

	if hostPartsLen == 1 {
		return host
	}

	if hostPartsLen == 2 {
		if hostParts[0] == "www" {
			return hostParts[1]
		} else {
			return host
		}
	}

	if hostPartsLen > 2 {
		return strings.Join(hostParts[hostPartsLen-2:], ".")
	}

	return host
}

// GetAppURL to get /app url if not configured by user
func GetAppURL(gc *gin.Context) string {
	return GetAppURLFromRequest(gc.Request)
}

// GetAppURLFromRequest is the transport-agnostic form of GetAppURL.
func GetAppURLFromRequest(r *http.Request) string {
	return GetHostFromRequest(r) + "/app"
}
