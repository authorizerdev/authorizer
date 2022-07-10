package validators

import "github.com/authorizerdev/authorizer/server/constants"

// IsValidWebhookEventName to validate webhook event name
func IsValidWebhookEventName(eventName string) bool {
	if eventName != constants.UserCreatedWebhookEvent && eventName != constants.UserLoginWebhookEvent && eventName != constants.UserSignUpWebhookEvent && eventName != constants.UserDeletedWebhookEvent && eventName != constants.UserAccessEnabledWebhookEvent && eventName != constants.UserAccessRevokedWebhookEvent {
		return false
	}

	return true
}
