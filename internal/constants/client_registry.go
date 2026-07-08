package constants

// Client.Kind discriminator values. Immutable after creation.
const (
	// ClientKindInteractive is a human-facing login client (browser/app
	// authorization_code + PKCE flows).
	ClientKindInteractive = "interactive"

	// ClientKindServiceAccount is a machine/workload client using the
	// client_credentials grant or workload identity federation.
	ClientKindServiceAccount = "service_account"
)

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
