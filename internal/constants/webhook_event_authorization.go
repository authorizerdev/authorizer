package constants

const (
	// ResourceCreatedWebhookEvent is fired when an authorization resource is created.
	ResourceCreatedWebhookEvent = "resource.created"
	// ResourceUpdatedWebhookEvent is fired when an authorization resource is updated.
	ResourceUpdatedWebhookEvent = "resource.updated"
	// ResourceDeletedWebhookEvent is fired when an authorization resource is deleted.
	ResourceDeletedWebhookEvent = "resource.deleted"
	// ScopeCreatedWebhookEvent is fired when an authorization scope is created.
	ScopeCreatedWebhookEvent = "scope.created"
	// ScopeUpdatedWebhookEvent is fired when an authorization scope is updated.
	ScopeUpdatedWebhookEvent = "scope.updated"
	// ScopeDeletedWebhookEvent is fired when an authorization scope is deleted.
	ScopeDeletedWebhookEvent = "scope.deleted"
	// PolicyCreatedWebhookEvent is fired when an authorization policy is created.
	PolicyCreatedWebhookEvent = "policy.created"
	// PolicyUpdatedWebhookEvent is fired when an authorization policy is updated.
	PolicyUpdatedWebhookEvent = "policy.updated"
	// PolicyDeletedWebhookEvent is fired when an authorization policy is deleted.
	PolicyDeletedWebhookEvent = "policy.deleted"
	// PermissionCreatedWebhookEvent is fired when an authorization permission is created.
	PermissionCreatedWebhookEvent = "permission.created"
	// PermissionUpdatedWebhookEvent is fired when an authorization permission is updated.
	PermissionUpdatedWebhookEvent = "permission.updated"
	// PermissionDeletedWebhookEvent is fired when an authorization permission is deleted.
	PermissionDeletedWebhookEvent = "permission.deleted"
	// PermissionCheckDeniedWebhookEvent is fired when a permission check is denied
	// in enforcing mode. Useful for agent kill-switches and security alerting.
	PermissionCheckDeniedWebhookEvent = "permission.check_denied"
)
