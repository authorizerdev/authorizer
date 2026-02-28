package parsers

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetHost returns the authorizer host URL from the request context.
// Priority: X-Authorizer-URL header, then scheme (X-Forwarded-Proto) + host (X-Forwarded-Host or Request.Host).
func GetHost(c *gin.Context) string {
	authorizerURL := c.Request.Header.Get("X-Authorizer-URL")
	if authorizerURL != "" {
		return strings.TrimSuffix(authorizerURL, "/")
	}

	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme != "https" {
		scheme = "http"
	}
	host := c.Request.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return strings.TrimSuffix(scheme+"://"+host, "/")
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
