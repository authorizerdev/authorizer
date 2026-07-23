package validators

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// REGRESSION: IsValidWebhookEventName's allow-list was never updated when
// PR #705 added the SCIM/group webhook events
// (internal/constants/webhook_event_scim.go) - registering a webhook for
// any of them was rejected as invalid, leaving the whole SCIM webhook
// feature with no working configuration path.
func TestIsValidWebhookEventName(t *testing.T) {
	valid := []string{
		constants.UserCreatedWebhookEvent,
		constants.UserLoginWebhookEvent,
		constants.UserSignUpWebhookEvent,
		constants.UserDeletedWebhookEvent,
		constants.UserAccessEnabledWebhookEvent,
		constants.UserAccessRevokedWebhookEvent,
		constants.UserDeactivatedWebhookEvent,
		constants.UserProvisionedWebhookEvent,
		constants.UserDeprovisionedWebhookEvent,
		constants.UserScimUpdatedWebhookEvent,
		constants.GroupCreatedWebhookEvent,
		constants.GroupUpdatedWebhookEvent,
		constants.GroupDeletedWebhookEvent,
	}
	for _, name := range valid {
		if !IsValidWebhookEventName(name) {
			t.Errorf("IsValidWebhookEventName(%q) = false, want true", name)
		}
	}

	invalid := []string{"", "not.a.real.event", "user.created "}
	for _, name := range invalid {
		if IsValidWebhookEventName(name) {
			t.Errorf("IsValidWebhookEventName(%q) = true, want false", name)
		}
	}
}
