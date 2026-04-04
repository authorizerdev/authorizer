package parsers

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetHost returns the authorizer host URL from the request context.
// Priority: X-Authorizer-URL header, then scheme (X-Forwarded-Proto) + host (X-Forwarded-Host or Request.Host).
// Headers are validated to prevent host header injection attacks.
func GetHost(c *gin.Context) string {
	authorizerURL := strings.TrimSpace(c.Request.Header.Get("X-Authorizer-URL"))
	if authorizerURL != "" {
		if sanitized := sanitizeAuthorizerURL(authorizerURL); sanitized != "" {
			return sanitized
		}
		// Invalid header value — fall through to standard host detection
	}

	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme != "https" {
		scheme = "http"
	}
	host := sanitizeHost(c.Request.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = sanitizeHost(c.Request.Host)
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

// GetHostName function returns hostname and port
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
	envAppURL := GetHost(gc) + "/app"
	return envAppURL
}
