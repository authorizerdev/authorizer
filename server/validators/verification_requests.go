package validators

import "github.com/authorizerdev/authorizer/server/constants"

// IsValidVerificationIdentifier validates verification identifier that is used to identify
// the type of verification request
func IsValidVerificationIdentifier(identifier string) bool {
	if identifier != constants.VerificationTypeBasicAuthSignup && identifier != constants.VerificationTypeForgotPassword && identifier != constants.VerificationTypeUpdateEmail {
		return false
	}
	return true
}
