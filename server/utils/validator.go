package utils

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/gin-gonic/gin"
)

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func IsValidOrigin(url string) bool {
	if len(constants.EnvData.ALLOWED_ORIGINS) == 1 && constants.EnvData.ALLOWED_ORIGINS[0] == "*" {
		return true
	}

	hasValidURL := false
	hostName, port := GetHostParts(url)
	currentOrigin := hostName + ":" + port

	for _, origin := range constants.EnvData.ALLOWED_ORIGINS {
		replacedString := origin
		// if has regex whitelisted domains
		if strings.Contains(origin, "*") {
			replacedString = strings.Replace(origin, ".", "\\.", -1)
			replacedString = strings.Replace(replacedString, "*", ".*", -1)

			if strings.HasPrefix(replacedString, ".*") {
				replacedString += "\\b"
			}

			if strings.HasSuffix(replacedString, ".*") {
				replacedString = "\\b" + replacedString
			}
		}

		if matched, _ := regexp.MatchString(replacedString, currentOrigin); matched {
			hasValidURL = true
			break
		}
	}

	return hasValidURL
}

func IsSuperAdmin(gc *gin.Context) bool {
	token, err := GetAdminAuthToken(gc)
	if err != nil {
		secret := gc.Request.Header.Get("x-authorizer-admin-secret")
		if secret == "" {
			return false
		}

		return secret == constants.EnvData.ADMIN_SECRET
	}

	return token != ""
}

func IsValidRoles(userRoles []string, roles []string) bool {
	valid := true
	for _, role := range roles {
		if !StringSliceContains(userRoles, role) {
			valid = false
			break
		}
	}

	return valid
}

func IsValidVerificationIdentifier(identifier string) bool {
	if identifier != enum.BasicAuthSignup.String() && identifier != enum.ForgotPassword.String() && identifier != enum.UpdateEmail.String() {
		return false
	}
	return true
}

func IsStringArrayEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
