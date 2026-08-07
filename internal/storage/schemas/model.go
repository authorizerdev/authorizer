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
	ScimGroup              string
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
		ScimGroup:              Prefix + "scim_groups",
		WebauthnCredential:     Prefix + "webauthn_credentials",
		OrgDomain:              Prefix + "org_domains",
		SAMLServiceProvider:    Prefix + "saml_service_providers",
		SAMLIDPKey:             Prefix + "saml_idp_keys",
	}

	// UserOwnedCollections are every collection keyed on `user_id` that a hard
	// delete of a user (StorageProvider.DeleteUser) MUST cascade to. This is the
	// single source of truth: all six backends iterate it, so a new user-keyed
	// table is covered everywhere by adding one line here.
	//
	// A missed entry is not cosmetic. An orphaned authorizer_federated_identities
	// row keeps pointing at a dead user id, jitProvisionFederatedUser fails
	// closed on it, and the (org_id, issuer, subject) uniqueness prevents
	// re-provisioning — a permanent SSO lockout for that principal.
	//
	// Soft deletes (DeactivateAccount, revoke access) only stamp the user row and
	// must NOT cascade — the account is meant to come back.
	UserOwnedCollections = []string{
		Collections.Session,
		Collections.FederatedIdentity,
		Collections.OrgMembership,
		Collections.Authenticators,
		Collections.WebauthnCredential,
		Collections.SessionToken,
		Collections.MFASession,
	}
)
