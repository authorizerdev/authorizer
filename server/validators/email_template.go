package validators

import "github.com/authorizerdev/authorizer/server/constants"

// IsValidEmailTemplateEventName function to validate email template events
func IsValidEmailTemplateEventName(eventName string) bool {
	if eventName != constants.VerificationTypeBasicAuthSignup && eventName != constants.VerificationTypeForgotPassword && eventName != constants.VerificationTypeMagicLinkLogin && eventName != constants.VerificationTypeUpdateEmail && eventName != constants.VerificationTypeOTP && eventName != constants.VerificationTypeInviteMember {
		return false
	}

	return true
}
