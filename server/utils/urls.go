package utils

import (
	"net/url"
	"strings"

	"github.com/yauthdev/yauth/server/constants"
)

func GetFrontendHost() string {
	u, err := url.Parse(constants.FRONTEND_URL)
	if err != nil {
		return `localhost`
	}

	host := u.Hostname()
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
