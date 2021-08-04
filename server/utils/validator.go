package utils

import (
	"net/mail"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
)

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func IsValidRedirectURL(url string) bool {
	if len(constants.ALLOWED_ORIGINS) == 1 && constants.ALLOWED_ORIGINS[0] == "*" {
		return true
	}

	hasValidURL := false
	urlDomain := GetDomainName(url)

	for _, val := range constants.ALLOWED_ORIGINS {
		if strings.Contains(val, urlDomain) {
			hasValidURL = true
			break
		}
	}

	return hasValidURL
}
