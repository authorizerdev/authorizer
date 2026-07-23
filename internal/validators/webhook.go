package validators

import "github.com/authorizerdev/authorizer/internal/constants"

// validWebhookEventNames are the only event names an admin may register a
// webhook against. PR #705 added the SCIM/group events
// (internal/constants/webhook_event_scim.go) but never added them here, so
// registering a webhook for any of them was rejected as invalid - the whole
// SCIM webhook feature had no working configuration path.
var validWebhookEventNames = map[string]bool{
	constants.UserCreatedWebhookEvent:       true,
	constants.UserLoginWebhookEvent:         true,
	constants.UserSignUpWebhookEvent:        true,
	constants.UserDeletedWebhookEvent:       true,
	constants.UserAccessEnabledWebhookEvent: true,
	constants.UserAccessRevokedWebhookEvent: true,
	constants.UserDeactivatedWebhookEvent:   true,
	constants.UserProvisionedWebhookEvent:   true,
	constants.UserDeprovisionedWebhookEvent: true,
	constants.UserScimUpdatedWebhookEvent:   true,
	constants.GroupCreatedWebhookEvent:      true,
	constants.GroupUpdatedWebhookEvent:      true,
	constants.GroupDeletedWebhookEvent:      true,
}

// IsValidWebhookEventName to validate webhook event name
func IsValidWebhookEventName(eventName string) bool {
	return validWebhookEventNames[eventName]
}
