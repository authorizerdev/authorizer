package constants

const (
	// Authentication events
	AuditLoginSuccessEvent     = "user.login_success"
	AuditLoginFailedEvent      = "user.login_failed"
	AuditSignupEvent           = "user.signup"
	AuditLogoutEvent           = "user.logout"
	AuditPasswordChangedEvent  = "user.password_changed"
	AuditPasswordResetEvent    = "user.password_reset"
	AuditEmailVerifiedEvent    = "user.email_verified"
	AuditPhoneVerifiedEvent    = "user.phone_verified"
	AuditMFAEnabledEvent       = "user.mfa_enabled"
	AuditMFADisabledEvent      = "user.mfa_disabled"
	AuditUserDeactivatedEvent  = "user.deactivated"

	// Admin events
	AuditAdminUserCreatedEvent   = "admin.user_created"
	AuditAdminUserUpdatedEvent   = "admin.user_updated"
	AuditAdminUserDeletedEvent   = "admin.user_deleted"
	AuditAdminAccessRevokedEvent = "admin.access_revoked"
	AuditAdminAccessEnabledEvent = "admin.access_enabled"
	AuditAdminUserUnlockedEvent  = "admin.user_unlocked"
	AuditAdminConfigChangedEvent = "admin.config_changed"
	AuditAdminWebhookCreated     = "admin.webhook_created"
	AuditAdminWebhookUpdated     = "admin.webhook_updated"
	AuditAdminWebhookDeleted     = "admin.webhook_deleted"

	// Token events
	AuditTokenIssuedEvent    = "token.issued"
	AuditTokenRefreshedEvent = "token.refreshed"
	AuditTokenRevokedEvent   = "token.revoked"

	// Session events
	AuditSessionCreatedEvent    = "session.created"
	AuditSessionTerminatedEvent = "session.terminated"
)
