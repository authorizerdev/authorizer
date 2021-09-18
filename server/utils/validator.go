package utils

import (
	"net/mail"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/gin-gonic/gin"
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

func IsSuperAdmin(gc *gin.Context) bool {
	secret := gc.Request.Header.Get("x-authorizer-admin-secret")
	if secret == "" {
		return false
	}

	return secret == constants.ADMIN_SECRET
}

func IsValidRolesArray(roles []*string) bool {
	valid := true
	currentRoleMap := map[string]bool{}

	for _, currentRole := range constants.ROLES {
		currentRoleMap[currentRole] = true
	}
	for _, inputRole := range roles {
		if !currentRoleMap[*inputRole] {
			valid = false
			break
		}
	}
	return valid
}
