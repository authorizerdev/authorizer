package constants

const (

	// UserLoginWebhookEvent name for login event
	UserLoginWebhookEvent = `user.login`
	// UserCreatedWebhookEvent name for user creation event
	// This is triggered when user entry is created but still not verified
	UserCreatedWebhookEvent = `user.created`
	// UserSignUpWebhookEvent name for signup event
	UserSignUpWebhookEvent = `user.signup`
	// UserAccessRevokedWebhookEvent name for user access revoke event
	UserAccessRevokedWebhookEvent = `user.access_revoked`
	// UserAccessEnabledWebhookEvent name for user access enable event
	UserAccessEnabledWebhookEvent = `user.access_enabled`
	// UserDeletedWebhookEvent name for user deleted event
	UserDeletedWebhookEvent = `user.deleted`
)
