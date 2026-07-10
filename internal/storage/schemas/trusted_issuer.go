package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TrustedIssuer registers an external JWT issuer whose tokens are accepted as
// client credentials for a Client (RFC 7523 client_assertion flow).
//
// One Client may have multiple TrustedIssuers (e.g. K8s SA tokens AND
// a SPIFFE JWT-SVID from the same workload). Each TrustedIssuer maps to exactly
// one Client.
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

	// Kind discriminates the trust relationship this row represents (design §4.3 /
	// §5 K1): constants.TrustKindClientAssertion (default), TrustKindSSOOIDC, or
	// TrustKindSSOSAML (reserved). Immutable after creation.
	//
	// SECURITY (design §5.2 CR1): the client_assertion resolver accepts ONLY
	// client_assertion_trust rows (with an empty OrgID). An sso_oidc row — which
	// has NO subject pin — must never be reachable on the client-authentication
	// path. A pre-existing row written before this column existed reads back as ""
	// and is interpreted as client_assertion_trust by EffectiveKind, so upgrades
	// don't break existing trust rows.
	Kind string `json:"kind" bson:"kind" cql:"kind" dynamo:"kind" gorm:"default:'client_assertion_trust'"`

	// OrgID scopes an SSO connection (sso_oidc/sso_saml) to one Organization.
	// EMPTY for client_assertion_trust rows (which are instance-global). Immutable.
	// Part of the (kind, org_id) lookup that finds an org's SSO connection.
	OrgID string `json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" gorm:"index" index:"org_id,hash"`

	// ClientID links this issuer to the Client it authenticates. Empty on
	// sso_oidc/sso_saml rows (which federate users, not clients). client_id is a
	// DynamoDB GSI key and DynamoDB rejects an empty-string key value, so
	// dynamo:"...,omitempty" omits it when empty — the row simply doesn't appear
	// in the client_id GSI (sparse index), which is correct for SSO rows.
	ClientID string `json:"client_id" bson:"client_id" cql:"client_id" dynamo:"client_id,omitempty" gorm:"index" index:"client_id,hash"`

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

	// AllowedSubjects is a comma-separated allow-list of the exact subject values
	// (the value of the SubjectClaim claim) permitted to authenticate as this
	// issuer's Client — e.g. "system:serviceaccount:prod:payments".
	//
	// SECURITY (design §5.2 C1): an empty AllowedSubjects is DENY-ALL, mirroring
	// Client.AllowedScopes. A row with no configured subjects authenticates
	// nobody; the client_assertion resolver MUST reject when the parsed list is
	// empty. Matching is EXACT (never prefix/substring) to defeat subject
	// confusion (H3): "prod-evil" must not satisfy a pin of "prod".
	AllowedSubjects string `json:"allowed_subjects" bson:"allowed_subjects" cql:"allowed_subjects" dynamo:"allowed_subjects"`

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

	// --- SSO OIDC broker (kind = sso_oidc) ---
	// These fields hold the upstream IdP configuration Authorizer uses as the
	// Relying Party. They are empty on client_assertion_trust rows.

	// SSOClientID is the client_id Authorizer was issued AT the upstream IdP.
	SSOClientID string `json:"sso_client_id" bson:"sso_client_id" cql:"sso_client_id" dynamo:"sso_client_id"`

	// SSOClientSecretEnc is the upstream client_secret, AES-encrypted at rest
	// (crypto.EncryptAES keyed on Config.ClientSecret — reversible because the
	// value is replayed to the upstream token endpoint; a bcrypt hash could not be
	// used). json:"-" so it NEVER serializes into an API/webhook/log projection.
	SSOClientSecretEnc string `json:"-" bson:"sso_client_secret_enc" cql:"sso_client_secret_enc" dynamo:"sso_client_secret_enc"`

	// SSOScopes is the space-separated scope set requested at the upstream IdP.
	// Defaults to "openid profile email" when empty.
	SSOScopes string `json:"sso_scopes" bson:"sso_scopes" cql:"sso_scopes" dynamo:"sso_scopes"`

	// SSORedirectURI is the redirect_uri registered at the upstream IdP that points
	// back at Authorizer's broker callback. When empty the broker derives
	// {scheme}://{host}/oauth/sso/{org}/callback from the request host.
	SSORedirectURI string `json:"sso_redirect_uri" bson:"sso_redirect_uri" cql:"sso_redirect_uri" dynamo:"sso_redirect_uri"`

	// --- SSO SAML SP (kind = sso_saml) ---
	// These fields hold the upstream corporate IdP configuration Authorizer uses
	// as a SAML 2.0 Service Provider. They are empty on non-sso_saml rows. The IdP
	// entity ID (the assertion Issuer and the FederatedIdentity issuer) reuses the
	// globally-unique IssuerURL field, so a SAML IdP cannot shadow an OIDC/client-
	// assertion issuer at the same value and vice-versa.

	// SAMLSSOURL is the upstream IdP Single Sign-On endpoint the signed
	// AuthnRequest is sent to (HTTP-Redirect binding).
	SAMLSSOURL *string `json:"saml_sso_url" bson:"saml_sso_url" cql:"saml_sso_url" dynamo:"saml_sso_url"`

	// SAMLIDPCertPEM is the IdP's X.509 signing certificate (PEM). Assertion
	// signatures are validated ONLY against this certificate — the sole trust
	// anchor for this org's connection.
	SAMLIDPCertPEM *string `json:"saml_idp_cert_pem" bson:"saml_idp_cert_pem" cql:"saml_idp_cert_pem" dynamo:"saml_idp_cert_pem"`

	// SAMLSPEntityID is the SP entity ID Authorizer advertises for this org (the
	// assertion Audience an incoming assertion MUST target). When empty the SP
	// derives {scheme}://{host}/oauth/saml/{org}/metadata from the request host.
	SAMLSPEntityID *string `json:"saml_sp_entity_id" bson:"saml_sp_entity_id" cql:"saml_sp_entity_id" dynamo:"saml_sp_entity_id"`

	// SAMLACSURL is the Assertion Consumer Service URL for this org (the
	// Recipient/Destination an incoming assertion MUST target). When empty the SP
	// derives {scheme}://{host}/oauth/saml/{org}/acs from the request host.
	SAMLACSURL *string `json:"saml_acs_url" bson:"saml_acs_url" cql:"saml_acs_url" dynamo:"saml_acs_url"`

	// SAMLAttributeMapping is a JSON object mapping Authorizer profile fields to
	// upstream SAML attribute names, e.g.
	// {"email":"email","given_name":"firstName","family_name":"lastName"}.
	// The NameID is ALWAYS the federated-identity subject; this map only governs
	// optional profile-attribute extraction. Empty means "use default names".
	SAMLAttributeMapping *string `json:"saml_attribute_mapping" bson:"saml_attribute_mapping" cql:"saml_attribute_mapping" dynamo:"saml_attribute_mapping"`

	// SAMLAllowIDPInitiated permits IdP-initiated SSO (a POST to ACS with no
	// matching pending AuthnRequest). DEFAULT FALSE — SP-initiated only, with
	// InResponseTo bound to a pending request. Enable only if the org's IdP does
	// not support SP-initiated flows and the operator accepts the reduced CSRF
	// protection. NOTE: enabling this disables InResponseTo validation for ALL
	// responses on this connection (crewjam limitation), including SP-initiated
	// ones — the assertion then relies solely on the single-use AssertionID cache
	// for replay defence.
	SAMLAllowIDPInitiated bool `json:"saml_allow_idp_initiated" bson:"saml_allow_idp_initiated" cql:"saml_allow_idp_initiated" dynamo:"saml_allow_idp_initiated"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// EffectiveKind returns the row's Kind, treating an empty value as
// TrustKindClientAssertion. A row written before the kind column existed reads
// back with an empty Kind; interpreting that as client_assertion_trust keeps
// existing trust rows working across an upgrade AND keeps the CR1 guard correct
// (an sso_oidc row always carries an explicit non-empty Kind, so it can never be
// mistaken for a client_assertion row).
func (t *TrustedIssuer) EffectiveKind() string {
	if strings.TrimSpace(t.Kind) == "" {
		return constants.TrustKindClientAssertion
	}
	return t.Kind
}

// ParsedAllowedSubjects returns AllowedSubjects as a slice: comma-separated,
// whitespace trimmed, empty segments dropped. This is the single source of
// truth for interpreting the stored subject allow-list — the client_assertion
// resolver uses it to exact-match the presented token's subject claim.
// An empty or whitespace-only AllowedSubjects yields an empty slice, which the
// resolver MUST treat as DENY-ALL (schema § AllowedSubjects comment).
func (t *TrustedIssuer) ParsedAllowedSubjects() []string {
	subjects := []string{}
	for _, s := range strings.Split(t.AllowedSubjects, ",") {
		if s = strings.TrimSpace(s); s != "" {
			subjects = append(subjects, s)
		}
	}
	return subjects
}

// AsAPITrustedIssuer converts the storage record into the GraphQL model. The
// Phase 4 Kubernetes TokenReview fields (EnableTokenReview, KubernetesAPIServerURL)
// are surfaced in the admin API; the remaining Phase 5/6 fields (SPIFFE bundle
// refresh, mTLS proxy) are intentionally not surfaced.
func (t *TrustedIssuer) AsAPITrustedIssuer() *model.TrustedIssuer {
	id := t.ID
	if strings.Contains(id, Collections.TrustedIssuer+"/") {
		id = strings.TrimPrefix(id, Collections.TrustedIssuer+"/")
	}
	return &model.TrustedIssuer{
		ID:                       id,
		ServiceAccountID:         t.ClientID,
		Name:                     t.Name,
		IssuerURL:                t.IssuerURL,
		KeySourceType:            t.KeySourceType,
		JwksURL:                  t.JWKSUrl,
		ExpectedAud:              t.ExpectedAud,
		SubjectClaim:             t.SubjectClaim,
		AllowedSubjects:          refs.NewStringRef(t.AllowedSubjects),
		IssuerType:               t.IssuerType,
		IsActive:                 t.IsActive,
		SpiffeRefreshHintSeconds: refs.NewInt64Ref(t.SpiffeRefreshHintSeconds),
		EnableTokenReview:        t.EnableTokenReview,
		KubernetesAPIServerURL:   t.KubernetesAPIServerURL,
		CreatedAt:                refs.NewInt64Ref(t.CreatedAt),
		UpdatedAt:                refs.NewInt64Ref(t.UpdatedAt),
	}
}
