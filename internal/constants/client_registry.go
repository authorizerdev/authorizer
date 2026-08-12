package constants

// Client.Kind discriminator values. Immutable after creation.
const (
	// ClientKindInteractive is a human-facing login client (browser/app
	// authorization_code + PKCE flows).
	ClientKindInteractive = "interactive"

	// ClientKindServiceAccount is a machine/workload client using the
	// client_credentials grant or workload identity federation.
	ClientKindServiceAccount = "service_account"

	// ClientKindDynamic is an interactive client that registered ITSELF through
	// the RFC 7591 endpoint, with no operator involvement. Behaviourally it is an
	// interactive client; the separate kind exists because its identity is
	// self-asserted, which is what drives the consent screen and the mandatory
	// PKCE check. A distinct Kind value rather than a new column keeps this off
	// the schema-change path across all six storage backends — every existing
	// Kind test is a positive `== ClientKindServiceAccount` comparison, so a new
	// value cannot silently widen an existing grant.
	ClientKindDynamic = "dynamic"
)

// IsSelfRegistered reports whether a client asserted its own identity rather
// than being vouched for by an operator. Such clients get the consent screen and
// are required to use PKCE. CIMD clients are the other self-asserted kind; they
// never reach the registry, so they are detected by their URL client_id instead.
func IsSelfRegistered(kind string) bool { return kind == ClientKindDynamic }

// Client.TokenEndpointAuthMethod values (RFC 7591 §2 / OIDC Core §9).
const (
	// TokenEndpointAuthMethodClientSecretBasic sends client_id/client_secret in
	// the HTTP Authorization header (RFC 6749 §2.3.1).
	TokenEndpointAuthMethodClientSecretBasic = "client_secret_basic"

	// TokenEndpointAuthMethodClientSecretPost sends client_id/client_secret in
	// the request body.
	TokenEndpointAuthMethodClientSecretPost = "client_secret_post"

	// TokenEndpointAuthMethodNone is a public client with no secret; it proves
	// possession via PKCE.
	TokenEndpointAuthMethodNone = "none"
)
