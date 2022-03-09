package utils

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
)

// IsValidEmail validates email
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsValidOrigin validates origin based on ALLOWED_ORIGINS
func IsValidOrigin(url string) bool {
	allowedOrigins := envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyAllowedOrigins)
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true
	}

	hasValidURL := false
	hostName, port := GetHostParts(url)
	currentOrigin := hostName + ":" + port

	for _, origin := range allowedOrigins {
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

// IsValidRoles validates roles
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

// IsValidVerificationIdentifier validates verification identifier that is used to identify
// the type of verification request
func IsValidVerificationIdentifier(identifier string) bool {
	if identifier != constants.VerificationTypeBasicAuthSignup && identifier != constants.VerificationTypeForgotPassword && identifier != constants.VerificationTypeUpdateEmail {
		return false
	}
	return true
}

// IsStringArrayEqual validates if string array are equal.
// This does check if the order is same
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
