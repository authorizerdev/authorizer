package constants

// OAuth2 grant type URNs for the token endpoint (RFC 6749 + RFC 8693).
const (
	// GrantTypeAuthorizationCode is the standard browser-redirect grant (RFC 6749 §4.1).
	GrantTypeAuthorizationCode = "authorization_code"

	// GrantTypeRefreshToken rotates an existing refresh token (RFC 6749 §6).
	GrantTypeRefreshToken = "refresh_token"

	// GrantTypeClientCredentials issues tokens for machine/service identities
	// without a human resource owner (RFC 6749 §4.4).
	// Authenticate using client_id (= Client.ID) and client_secret,
	// or via client_assertion for workload identity federation (Phases 3–5).
	GrantTypeClientCredentials = "client_credentials"

	// GrantTypeTokenExchange enables on-behalf-of delegation and workload-to-user
	// token exchange (RFC 8693).
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
)

// OAuth2 client assertion type URNs (RFC 7521 / RFC 7523).
const (
	// ClientAssertionTypeJWTBearer identifies a JWT as the client credential
	// in the client_assertion parameter (RFC 7523).
	// Used for Kubernetes SA tokens and generic OIDC workload tokens (Phases 3–4).
	ClientAssertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

	// ClientAssertionTypeJWTSPIFFE identifies a SPIFFE JWT-SVID as the client
	// credential (draft-schwenkschuster-oauth-spiffe-client-auth-00).
	// PREVIEW: the underlying draft expired 2026-01-02 and is not WG-adopted.
	// This URN is not IANA-registered and may change. Ship as experimental only.
	ClientAssertionTypeJWTSPIFFE = "urn:ietf:params:oauth:client-assertion-type:jwt-spiffe"
)

// RFC 8693 token type URNs. Only the two this server ACCEPTS are listed — see
// isSupportedExchangeTokenType. The refresh_token and id_token URNs the RFC also
// defines were declared here and never referenced; add them back alongside the
// code that accepts them, so the list stays a statement about this server rather
// than a copy of the registry.
const (
	// TokenTypeURNAccessToken identifies an OAuth2 access token.
	TokenTypeURNAccessToken = "urn:ietf:params:oauth:token-type:access_token"

	// TokenTypeURNJWT identifies a generic JWT.
	TokenTypeURNJWT = "urn:ietf:params:oauth:token-type:jwt"
)

// TrustedIssuer key source types controlling how JWKS are fetched.
const (
	// KeySourceOIDCDiscovery fetches the JWKS URI from the issuer's OpenID
	// Connect discovery document (/.well-known/openid-configuration).
	// Not suitable for private K8s clusters that do not expose discovery publicly.
	KeySourceOIDCDiscovery = "oidc_discovery"

	// KeySourceStaticJWKSURL fetches JWKS directly from a configured URL.
	// Preferred for private clusters — avoids exposing K8s discovery endpoints.
	KeySourceStaticJWKSURL = "static_jwks_url"

	// KeySourceSPIFFEBundleEndpoint names a SPIFFE bundle endpoint as the key
	// source.
	//
	// NOT IMPLEMENTED. fetchJWKSBytes has no case for it and returns
	// "unsupported key_source_type", and SpiffeRefreshHintSeconds — which a
	// bundle endpoint's refresh cadence depends on — is stored and returned by
	// the admin API but never read at runtime.
	//
	// The constant is kept, rather than deleted, because
	// service.validateKeySourceType rejects it BY NAME with "not implemented
	// yet". Before that existed the value was accepted and stored verbatim, so an
	// issuer configured with it looked healthy and failed only when the first
	// workload tried to authenticate.
	KeySourceSPIFFEBundleEndpoint = "spiffe_bundle_endpoint"
)

// TrustedIssuer issuer type identifiers.
const (
	// IssuerTypeKubernetesSA identifies a Kubernetes projected ServiceAccount token.
	IssuerTypeKubernetesSA = "kubernetes_sa"

	// IssuerTypeSPIFFEJWT identifies a SPIFFE JWT-SVID (Phase 5, preview).
	IssuerTypeSPIFFEJWT = "spiffe_jwt"

	// IssuerTypeOIDC identifies a generic OIDC token from an external IdP.
	IssuerTypeOIDC = "oidc"

	// IssuerTypeCloudOIDC identifies a cloud-provider workload identity token
	// (AWS IRSA, GCP Workload Identity, Azure Managed Identity).
	IssuerTypeCloudOIDC = "cloud_oidc"
)

// TrustedIssuer authentication method identifiers.
const (
	// AuthMethodJWTAssertion uses a JWT as the client_assertion. It is the only
	// value AddTrustedIssuer writes, and the only one anything reads.
	AuthMethodJWTAssertion = "jwt_assertion"
)

// An x509_mtls auth method (X.509-SVID over mTLS) was declared here and never
// implemented: nothing read it, AddTrustedIssuer hardcoded jwt_assertion, and no
// request field could set it, so the constant was unreachable in every direction.
// Removed rather than left as a claim the code does not honour. Reintroduce it
// with the implementation, not ahead of it.
