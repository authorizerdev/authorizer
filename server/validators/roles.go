package validators

import "github.com/authorizerdev/authorizer/server/utils"

// IsValidRoles validates roles
func IsValidRoles(userRoles []string, roles []string) bool {
	valid := true
	for _, userRole := range userRoles {
		if !utils.StringSliceContains(roles, userRole) {
			valid = false
			break
		}
	}

	return valid
}
