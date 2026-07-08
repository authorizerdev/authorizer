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

// RFC 8693 token type URNs used in subject_token_type and issued_token_type.
const (
	// TokenTypeURNAccessToken identifies an OAuth2 access token.
	TokenTypeURNAccessToken = "urn:ietf:params:oauth:token-type:access_token"

	// TokenTypeURNRefreshToken identifies an OAuth2 refresh token.
	TokenTypeURNRefreshToken = "urn:ietf:params:oauth:token-type:refresh_token"

	// TokenTypeURNIDToken identifies an OpenID Connect ID token.
	TokenTypeURNIDToken = "urn:ietf:params:oauth:token-type:id_token"

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

	// KeySourceSPIFFEBundleEndpoint fetches keys from a SPIFFE bundle endpoint.
	// Requires SpiffeRefreshHintSeconds to be honoured at runtime (Phase 5).
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
	// AuthMethodJWTAssertion uses a JWT as the client_assertion (Phases 3–5, default).
	AuthMethodJWTAssertion = "jwt_assertion"

	// AuthMethodX509MTLS uses an X.509-SVID via mTLS (Phase 6).
	AuthMethodX509MTLS = "x509_mtls"
)
