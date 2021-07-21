package utils

import (
	"log"
	"net/url"

	"github.com/yauthdev/yauth/server/constants"
)

func GetFrontendHost() string {
	u, err := url.Parse(constants.FRONTEND_URL)
	if err != nil {
		return `localhost`
	}

	log.Println("hostname", "."+u.Hostname())

	return "." + u.Hostname()
}
