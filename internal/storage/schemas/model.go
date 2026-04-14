package schemas

// CollectionList / Tables available for authorizer in the database
type CollectionList struct {
	User                   string
	VerificationRequest    string
	Session                string
	Env                    string
	Webhook                string
	WebhookLog             string
	EmailTemplate          string
	OTP                    string
	SMSVerificationRequest string
	Authenticators         string
	SessionToken           string
	MFASession             string
	OAuthState             string
	AuditLog               string
	// Authorization tables
	Resource         string
	Scope            string
	Policy           string
	PolicyTarget     string
	Permission       string
	PermissionScope  string
	PermissionPolicy string
}

var (
	// Prefix for table name / collection names
	Prefix = "authorizer_"
	// Collections / Tables available for authorizer in the database (used for dbs other than gorm)
	Collections = CollectionList{
		User:                   Prefix + "users",
		VerificationRequest:    Prefix + "verification_requests",
		Session:                Prefix + "sessions",
		Env:                    Prefix + "env",
		Webhook:                Prefix + "webhooks",
		WebhookLog:             Prefix + "webhook_logs",
		EmailTemplate:          Prefix + "email_templates",
		OTP:                    Prefix + "otps",
		SMSVerificationRequest: Prefix + "sms_verification_requests",
		Authenticators:         Prefix + "authenticators",
		SessionToken:           Prefix + "session_tokens",
		MFASession:             Prefix + "mfa_sessions",
		OAuthState:             Prefix + "oauth_states",
		AuditLog:               Prefix + "audit_logs",
		// Authorization collections
		Resource:         Prefix + "resources",
		Scope:            Prefix + "scopes",
		Policy:           Prefix + "policies",
		PolicyTarget:     Prefix + "policy_targets",
		Permission:       Prefix + "permissions",
		PermissionScope:  Prefix + "permission_scopes",
		PermissionPolicy: Prefix + "permission_policies",
	}
)
