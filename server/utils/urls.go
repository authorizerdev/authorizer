package utils

import (
	"net/url"

	"github.com/yauthdev/yauth/server/constants"
)

func GetFrontendHost() string {
	u, err := url.Parse(constants.FRONTEND_URL)
	if err != nil {
		return `localhost`
	}

	return u.Hostname()
}
