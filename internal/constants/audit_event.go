package constants

// Audit event type constants used for structured audit logging.
// Each constant represents a specific auditable action in the system,
// organized by domain: user authentication, admin operations, OAuth,
// token lifecycle, and session management.
const (
	// AuditLoginSuccessEvent is logged when a user successfully authenticates.
	AuditLoginSuccessEvent = "user.login_success"
	// AuditLoginFailedEvent is logged when a user authentication attempt fails.
	AuditLoginFailedEvent = "user.login_failed"
	// AuditSignupEvent is logged when a new user registers.
	AuditSignupEvent = "user.signup"
	// AuditLogoutEvent is logged when a user logs out.
	AuditLogoutEvent = "user.logout"
	// AuditPasswordChangedEvent is logged when a user changes their password.
	AuditPasswordChangedEvent = "user.password_changed"
	// AuditPasswordResetEvent is logged when a user resets their password via token or OTP.
	AuditPasswordResetEvent = "user.password_reset"
	// AuditForgotPasswordEvent is logged when a user requests a password reset.
	AuditForgotPasswordEvent = "user.forgot_password_requested"
	// AuditMagicLinkRequestedEvent is logged when a user requests a magic link login.
	AuditMagicLinkRequestedEvent = "user.magic_link_requested"
	// AuditEmailVerifiedEvent is logged when a user's email is verified.
	AuditEmailVerifiedEvent = "user.email_verified"
	// AuditPhoneVerifiedEvent is logged when a user's phone number is verified.
	AuditPhoneVerifiedEvent = "user.phone_verified"
	// AuditMFAEnabledEvent is logged when a user enables multi-factor authentication.
	AuditMFAEnabledEvent = "user.mfa_enabled"
	// AuditMFADisabledEvent is logged when a user disables multi-factor authentication.
	AuditMFADisabledEvent = "user.mfa_disabled"
	// AuditProfileUpdatedEvent is logged when a user updates their profile.
	AuditProfileUpdatedEvent = "user.profile_updated"
	// AuditUserDeactivatedEvent is logged when a user deactivates their account.
	AuditUserDeactivatedEvent = "user.deactivated"
	// AuditOTPResentEvent is logged when an OTP is resent to a user.
	AuditOTPResentEvent = "user.otp_resent"
	// AuditVerifyEmailResentEvent is logged when a verification email is resent.
	AuditVerifyEmailResentEvent = "user.verify_email_resent"

	// AuditAdminLoginSuccessEvent is logged when an admin successfully authenticates.
	AuditAdminLoginSuccessEvent = "admin.login_success"
	// AuditAdminLoginFailedEvent is logged when an admin authentication attempt fails.
	AuditAdminLoginFailedEvent = "admin.login_failed"
	// AuditAdminLogoutEvent is logged when an admin logs out.
	AuditAdminLogoutEvent = "admin.logout"
	// AuditAdminUserCreatedEvent is logged when an admin creates a user.
	AuditAdminUserCreatedEvent = "admin.user_created"
	// AuditAdminUserUpdatedEvent is logged when an admin updates a user.
	AuditAdminUserUpdatedEvent = "admin.user_updated"
	// AuditAdminUserDeletedEvent is logged when an admin deletes a user.
	AuditAdminUserDeletedEvent = "admin.user_deleted"
	// AuditAdminAccessRevokedEvent is logged when an admin revokes a user's access.
	AuditAdminAccessRevokedEvent = "admin.access_revoked"
	// AuditAdminAccessEnabledEvent is logged when an admin restores a user's access.
	AuditAdminAccessEnabledEvent = "admin.access_enabled"
	// AuditAdminInviteSentEvent is logged when an admin sends a user invitation.
	AuditAdminInviteSentEvent = "admin.invite_sent"
	// AuditAdminConfigChangedEvent is logged when an admin modifies server configuration.
	AuditAdminConfigChangedEvent = "admin.config_changed"
	// AuditAdminWebhookCreatedEvent is logged when an admin creates a webhook.
	AuditAdminWebhookCreatedEvent = "admin.webhook_created"
	// AuditAdminWebhookUpdatedEvent is logged when an admin updates a webhook.
	AuditAdminWebhookUpdatedEvent = "admin.webhook_updated"
	// AuditAdminWebhookDeletedEvent is logged when an admin deletes a webhook.
	AuditAdminWebhookDeletedEvent = "admin.webhook_deleted"
	// AuditAdminEmailTemplateCreatedEvent is logged when an admin creates an email template.
	AuditAdminEmailTemplateCreatedEvent = "admin.email_template_created"
	// AuditAdminEmailTemplateUpdatedEvent is logged when an admin updates an email template.
	AuditAdminEmailTemplateUpdatedEvent = "admin.email_template_updated"
	// AuditAdminEmailTemplateDeletedEvent is logged when an admin deletes an email template.
	AuditAdminEmailTemplateDeletedEvent = "admin.email_template_deleted"

	// AuditOAuthLoginInitiatedEvent is logged when an OAuth login flow is started.
	AuditOAuthLoginInitiatedEvent = "oauth.login_initiated"
	// AuditOAuthCallbackSuccessEvent is logged when an OAuth callback completes successfully.
	AuditOAuthCallbackSuccessEvent = "oauth.callback_success"
	// AuditOAuthCallbackFailedEvent is logged when an OAuth callback fails.
	AuditOAuthCallbackFailedEvent = "oauth.callback_failed"

	// AuditTokenIssuedEvent is logged when a new token is issued.
	AuditTokenIssuedEvent = "token.issued"
	// AuditTokenRefreshedEvent is logged when a token is refreshed.
	AuditTokenRefreshedEvent = "token.refreshed"
	// AuditTokenRevokedEvent is logged when a token is revoked.
	AuditTokenRevokedEvent = "token.revoked"

	// AuditSessionCreatedEvent is logged when a new session is created.
	AuditSessionCreatedEvent = "session.created"
	// AuditSessionTerminatedEvent is logged when a session is terminated.
	AuditSessionTerminatedEvent = "session.terminated"
)
