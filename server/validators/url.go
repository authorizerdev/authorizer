package validators

import (
	"regexp"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
)

// IsValidOrigin validates origin based on ALLOWED_ORIGINS
func IsValidOrigin(url string) bool {
	allowedOriginsString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAllowedOrigins)
	allowedOrigins := []string{}
	if err != nil {
		allowedOrigins = []string{"*"}
	} else {
		allowedOrigins = strings.Split(allowedOriginsString, ",")
	}
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true
	}

	hasValidURL := false
	hostName, port := parsers.GetHostParts(url)
	currentOrigin := hostName + ":" + port

	for _, origin := range allowedOrigins {
		replacedString := origin
		// if has regex whitelisted domains
		if strings.Contains(origin, "*") {
			replacedString = strings.ReplaceAll(origin, ".", "\\.")
			replacedString = strings.ReplaceAll(replacedString, "*", ".*")

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
