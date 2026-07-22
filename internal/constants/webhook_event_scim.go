package constants

// SCIM provisioning-lifecycle webhook events (RFC 7644 directory sync). Fired
// by the inbound SCIM server so a customer's own audit/automation pipeline can
// react to IdP-driven changes. User events carry a `user` payload; group events
// carry a `group` payload (see events.RegisterScimGroupEvent).
const (
	// UserProvisionedWebhookEvent fires when an IdP creates a user via SCIM.
	UserProvisionedWebhookEvent = `user.provisioned`
	// UserDeprovisionedWebhookEvent fires when an IdP deactivates a user via SCIM
	// (active:false PATCH/PUT or DELETE).
	UserDeprovisionedWebhookEvent = `user.deprovisioned`
	// UserScimUpdatedWebhookEvent fires when an IdP changes a user's attributes
	// via SCIM (name/email/phone/externalId, or reactivation).
	UserScimUpdatedWebhookEvent = `user.scim_updated`
	// GroupCreatedWebhookEvent fires when an IdP creates a group via SCIM.
	GroupCreatedWebhookEvent = `group.created`
	// GroupUpdatedWebhookEvent fires when an IdP changes a group via SCIM
	// (displayName/externalId or membership).
	GroupUpdatedWebhookEvent = `group.updated`
	// GroupDeletedWebhookEvent fires when an IdP deletes a group via SCIM.
	GroupDeletedWebhookEvent = `group.deleted`
)
