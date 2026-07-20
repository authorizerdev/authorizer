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
	Client                 string
	TrustedIssuer          string
	Organization           string
	OrgMembership          string
	FederatedIdentity      string
	ScimEndpoint           string
	WebauthnCredential     string
	OrgDomain              string
	SAMLServiceProvider    string
	SAMLIDPKey             string
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
		Client:                 Prefix + "clients",
		TrustedIssuer:          Prefix + "trusted_issuers",
		Organization:           Prefix + "organizations",
		OrgMembership:          Prefix + "org_memberships",
		FederatedIdentity:      Prefix + "federated_identities",
		ScimEndpoint:           Prefix + "scim_endpoints",
		WebauthnCredential:     Prefix + "webauthn_credentials",
		OrgDomain:              Prefix + "org_domains",
		SAMLServiceProvider:    Prefix + "saml_service_providers",
		SAMLIDPKey:             Prefix + "saml_idp_keys",
	}
)
