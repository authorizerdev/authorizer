package utils

import (
	"net/url"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/gin-gonic/gin"
)

// GetHost returns hostname from request context
// if X-Authorizer-URL header is set it is given highest priority
// if EnvKeyAuthorizerURL is set it is given second highest priority.
// if above 2 are not set the requesting host name is used
func GetHost(c *gin.Context) string {
	authorizerURL := c.Request.Header.Get("X-Authorizer-URL")
	if authorizerURL != "" {
		return authorizerURL
	}

	authorizerURL = memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAuthorizerURL)
	if authorizerURL != "" {
		return authorizerURL
	}

	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme != "https" {
		scheme = "http"
	}
	host := c.Request.Host
	return scheme + "://" + host
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

// GetAppURL to get /app/ url if not configured by user
func GetAppURL(gc *gin.Context) string {
	envAppURL := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppURL)
	if envAppURL == "" {
		envAppURL = GetHost(gc) + "/app"
	}
	return envAppURL
}
