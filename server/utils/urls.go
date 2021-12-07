package utils

import (
	"net/url"
	"strings"
)

// GetHostName function to get hostname
func GetHostName(auth_url string) string {
	u, err := url.Parse(auth_url)
	if err != nil {
		return `localhost`
	}

	host := u.Hostname()

	return host
}

// function to get domain name
func GetDomainName(auth_url string) string {
	u, err := url.Parse(auth_url)
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
