package schemas

// TrustedIssuer registers an external JWT issuer whose tokens are accepted as
// client credentials for a ServiceAccount (RFC 7523 client_assertion flow).
//
// One ServiceAccount may have multiple TrustedIssuers (e.g. K8s SA tokens AND
// a SPIFFE JWT-SVID from the same workload). Each TrustedIssuer maps to exactly
// one ServiceAccount.
//
// Supported issuer types (IssuerType field):
//   - "kubernetes_sa"  — Kubernetes projected ServiceAccount tokens (Phase 4)
//   - "spiffe_jwt"     — SPIFFE JWT-SVIDs (Phase 5, preview)
//   - "oidc"           — Generic OIDC / cloud-provider tokens (Phase 3)
//   - "cloud_oidc"     — AWS/GCP/Azure workload identity tokens (Phase 3)
//
// Key-source types (KeySourceType field):
//   - "oidc_discovery"         — fetch JWKS via <IssuerURL>/.well-known/openid-configuration
//   - "static_jwks_url"        — fetch JWKS directly from JWKSUrl (required for private clusters)
//   - "spiffe_bundle_endpoint" — fetch from a SPIFFE bundle endpoint (Phase 5)
//
// Authentication methods (AuthMethod field):
//   - "jwt_assertion" — RFC 7523 client_assertion JWT (Phases 3–5, default)
//   - "x509_mtls"    — SPIFFE X.509-SVID via mTLS (Phase 6)
//
// Security invariants enforced at the service layer (not here):
//
//	S1  — aud claim in presented token MUST equal ExpectedAud.
//	S2  — alg allow-list: RS/ES/PS 256/384/512 only; none and HS* rejected.
//	S3  — exp claim MUST be present and valid.
//	S7  — offline JWKS validation does not confirm the bound K8s object still
//	      exists; EnableTokenReview provides online hardening for Phase 4.
//	S10 — static_jwks_url is always available; OIDC discovery is never forced.
//
// Note: any field addition must also be reflected in the cassandradb provider.
type TrustedIssuer struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// ServiceAccountID links this issuer to the ServiceAccount it authenticates.
	ServiceAccountID string `json:"service_account_id" bson:"service_account_id" cql:"service_account_id" dynamo:"service_account_id" gorm:"index" index:"service_account_id,hash"`

	// Name is a human-readable label (e.g. "prod-k8s-cluster").
	Name string `json:"name" bson:"name" cql:"name" dynamo:"name"`

	// IssuerURL is the `iss` claim value expected in presented JWTs.
	// For SPIFFE issuers this is the trust domain URI (e.g. "spiffe://example.org").
	// Unique per Authorizer instance — one URL maps to one TrustedIssuer.
	IssuerURL string `gorm:"uniqueIndex" json:"issuer_url" bson:"issuer_url" cql:"issuer_url" dynamo:"issuer_url" index:"issuer_url,hash"`

	// KeySourceType determines how the JWKS key set is fetched. See type-level docs.
	KeySourceType string `json:"key_source_type" bson:"key_source_type" cql:"key_source_type" dynamo:"key_source_type"`

	// JWKSUrl is required when KeySourceType is "static_jwks_url".
	// Strongly preferred over "oidc_discovery" for private K8s clusters (S10).
	JWKSUrl *string `json:"jwks_url" bson:"jwks_url" cql:"jwks_url" dynamo:"jwks_url"`

	// ExpectedAud is the audience value the presented token MUST contain (S1).
	// For K8s SA tokens this MUST be set to Authorizer's own issuer URL so that
	// tokens minted for other services cannot be replayed here.
	ExpectedAud string `json:"expected_aud" bson:"expected_aud" cql:"expected_aud" dynamo:"expected_aud"`

	// SubjectClaim is the JWT claim used to identify the workload. Defaults to "sub".
	SubjectClaim string `json:"subject_claim" bson:"subject_claim" cql:"subject_claim" dynamo:"subject_claim"`

	// IssuerType categorises the issuer. See type-level docs for valid values.
	IssuerType string `json:"issuer_type" bson:"issuer_type" cql:"issuer_type" dynamo:"issuer_type"`

	// AuthMethod selects the client-authentication mechanism. See type-level docs.
	// Defaults to "jwt_assertion".
	AuthMethod string `json:"auth_method" bson:"auth_method" cql:"auth_method" dynamo:"auth_method" gorm:"default:'jwt_assertion'"`

	// IsActive controls whether this issuer is accepted for authentication.
	IsActive bool `json:"is_active" bson:"is_active" cql:"is_active" dynamo:"is_active" gorm:"default:true"`

	// --- Phase 4: Kubernetes SA online validation ---

	// EnableTokenReview activates online K8s TokenReview validation (kubernetes_sa only).
	// When true, Authorizer calls the K8s API server after offline JWT verification to
	// confirm the bound Pod/ServiceAccount still exists.
	// Default false — offline JWKS validation only (same as Keycloak default).
	// SECURITY NOTE (S7): offline-only validation accepts tokens for deleted objects
	// until exp. Enable this flag for high-security workloads.
	EnableTokenReview bool `json:"enable_token_review" bson:"enable_token_review" cql:"enable_token_review" dynamo:"enable_token_review"`

	// KubernetesAPIServerURL is required when EnableTokenReview is true.
	// Example: "https://kubernetes.default.svc:443"
	KubernetesAPIServerURL *string `json:"kubernetes_api_server_url" bson:"kubernetes_api_server_url" cql:"kubernetes_api_server_url" dynamo:"kubernetes_api_server_url"`

	// --- Phase 5: SPIFFE bundle refresh ---

	// SpiffeRefreshHintSeconds controls the JWKS refresh cadence for
	// "spiffe_bundle_endpoint" key sources. Default 300 (5 min) when 0.
	// HARD RUNTIME REQUIREMENT: failure to refresh at this cadence will reject
	// valid SVIDs after a trust-bundle key rotation.
	SpiffeRefreshHintSeconds int64 `json:"spiffe_refresh_hint_seconds" bson:"spiffe_refresh_hint_seconds" cql:"spiffe_refresh_hint_seconds" dynamo:"spiffe_refresh_hint_seconds"`

	// --- Phase 6: X.509-SVID mTLS proxy forwarding ---

	// TrustedProxyHeader is the HTTP header name from which to read a
	// forwarded client certificate (PEM or DER base64) in Model B deployments.
	// Empty means direct TLS only (Model A).
	TrustedProxyHeader *string `json:"trusted_proxy_header" bson:"trusted_proxy_header" cql:"trusted_proxy_header" dynamo:"trusted_proxy_header"`

	// TrustedProxyCIDRs is a comma-separated list of CIDR ranges whose requests
	// are permitted to supply TrustedProxyHeader.
	// MANDATORY when TrustedProxyHeader is set — requests from outside this list
	// that carry the header MUST be rejected to prevent certificate spoofing.
	TrustedProxyCIDRs *string `json:"trusted_proxy_cidrs" bson:"trusted_proxy_cidrs" cql:"trusted_proxy_cidrs" dynamo:"trusted_proxy_cidrs"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
