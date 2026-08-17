package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// defaultSubjectClaim is the JWT claim used to identify the workload when the
// admin does not specify one (spec §6 AddTrustedIssuerRequest comment).
const defaultSubjectClaim = "sub"

// normalizeAPIServerURL trims the admin-supplied kubernetes_api_server_url and
// returns nil for an empty value so the column persists as NULL.
func normalizeAPIServerURL(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// validateTokenReviewConfig enforces the write-time invariants for the
// Kubernetes TokenReview config. The URL is fetched server-side at runtime, so
// even though this mutation is super-admin only we validate it is a well-formed
// https URL and require it whenever online TokenReview is enabled (fail-closed:
// enabling review without a reachable apiserver would silently reject every
// token at runtime). apiServerURL is expected to be already normalized.
// validateKeySourceType allow-lists key_source_type against the sources
// fetchJWKSBytes can actually resolve.
//
// Both fields used to be checked for non-emptiness only, so any string was
// stored verbatim. That mattered most for "spiffe_bundle_endpoint": it is a
// declared constant with no implementation — fetchJWKSBytes has no case for it
// and falls through to `unsupported key_source_type` — so an operator could
// create an issuer, receive a 200 and a row that looked configured, and discover
// it was dead only when the first workload tried to authenticate. A plain typo
// ("static_jwks_urls") failed exactly the same way, at exactly the same
// unhelpful moment.
//
// The set here is deliberately the set fetchJWKSBytes IMPLEMENTS, not the set of
// declared constants. When spiffe_bundle_endpoint is implemented, this is the
// second place to change, and the error message below is the reminder.
func validateKeySourceType(v string) error {
	switch strings.TrimSpace(v) {
	case "":
		return InvalidArgument("key_source_type is required")
	case constants.KeySourceOIDCDiscovery, constants.KeySourceStaticJWKSURL:
		return nil
	case constants.KeySourceSPIFFEBundleEndpoint:
		return InvalidArgument(fmt.Sprintf(
			"key_source_type %q is not implemented yet; use %q or %q",
			constants.KeySourceSPIFFEBundleEndpoint,
			constants.KeySourceOIDCDiscovery, constants.KeySourceStaticJWKSURL))
	default:
		return InvalidArgument(fmt.Sprintf(
			"unsupported key_source_type %q; supported values are %q and %q",
			v, constants.KeySourceOIDCDiscovery, constants.KeySourceStaticJWKSURL))
	}
}

// validateIssuerType allow-lists issuer_type.
//
// Unlike key_source_type this field drives no behaviour today — nothing switches
// on it, which is why every IssuerType* constant is otherwise unreferenced. It is
// still validated, because it is stored, returned by the admin API and shown in
// the dashboard: an unconstrained free-text column that looks like an enum will
// diverge across deployments and cannot later be given meaning without a
// migration. Rejecting a typo now is cheaper than reconciling one later.
func validateIssuerType(v string) error {
	switch strings.TrimSpace(v) {
	case "":
		return InvalidArgument("issuer_type is required")
	case constants.IssuerTypeKubernetesSA, constants.IssuerTypeSPIFFEJWT,
		constants.IssuerTypeOIDC, constants.IssuerTypeCloudOIDC:
		return nil
	default:
		return InvalidArgument(fmt.Sprintf(
			"unsupported issuer_type %q; supported values are %q, %q, %q and %q",
			v, constants.IssuerTypeKubernetesSA, constants.IssuerTypeSPIFFEJWT,
			constants.IssuerTypeOIDC, constants.IssuerTypeCloudOIDC))
	}
}

func validateTokenReviewConfig(enableTokenReview bool, apiServerURL *string) error {
	raw := refs.StringValue(apiServerURL)
	if raw == "" {
		if enableTokenReview {
			return InvalidArgument("kubernetes_api_server_url is required when enable_token_review is true")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return InvalidArgument(fmt.Sprintf("kubernetes_api_server_url is not a valid URL: %v", err))
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return InvalidArgument("kubernetes_api_server_url must be an https URL")
	}
	if u.Host == "" {
		return InvalidArgument("kubernetes_api_server_url must include a host")
	}
	return nil
}

// AddTrustedIssuer registers an external JWT issuer for a service account.
// subject_claim defaults to "sub" when omitted. Requires super-admin auth.
func (p *provider) AddTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.AddTrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddTrustedIssuer").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(params.ServiceAccountID) == "" {
		return nil, nil, InvalidArgument("service_account_id is required")
	}
	if strings.TrimSpace(params.Name) == "" {
		return nil, nil, InvalidArgument("name is required")
	}
	if strings.TrimSpace(params.IssuerURL) == "" {
		return nil, nil, InvalidArgument("issuer_url is required")
	}
	if err := validateKeySourceType(params.KeySourceType); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(params.ExpectedAud) == "" {
		return nil, nil, InvalidArgument("expected_aud is required")
	}
	if err := validateIssuerType(params.IssuerType); err != nil {
		return nil, nil, err
	}

	// Reject issuers bound to a non-existent service account — otherwise a typo
	// creates an orphan that can never be reached via the parent.
	if _, err := p.StorageProvider.GetClientByID(ctx, params.ServiceAccountID); err != nil {
		log.Debug().Err(err).Str("service_account_id", params.ServiceAccountID).Msg("service account not found")
		return nil, nil, NotFound(fmt.Sprintf("service account not found: %s", params.ServiceAccountID))
	}

	// Enforce issuer_url uniqueness at the service layer. Storage providers use a
	// plain insert with no unique constraint, so duplicates would otherwise
	// coexist and GetTrustedIssuerByIssuerURL (called on every client_assertion
	// validation) would resolve nondeterministically. This guard protects all
	// backends uniformly.
	if existing, err := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, params.IssuerURL); err == nil && existing != nil {
		log.Debug().Str("issuer_url", params.IssuerURL).Msg("issuer_url already registered")
		return nil, nil, AlreadyExists(fmt.Sprintf("issuer_url already registered: %s", params.IssuerURL))
	}

	subjectClaim := defaultSubjectClaim
	if params.SubjectClaim != nil && strings.TrimSpace(*params.SubjectClaim) != "" {
		subjectClaim = strings.TrimSpace(*params.SubjectClaim)
	}

	// AllowedSubjects is stored verbatim (trimmed); ParsedAllowedSubjects() is the
	// single interpreter. An empty/omitted value persists as "" — DENY-ALL by
	// design, so a freshly created issuer authenticates nobody until subjects are
	// explicitly configured (§5.2 C1, fail-closed).
	allowedSubjects := ""
	if params.AllowedSubjects != nil {
		allowedSubjects = strings.TrimSpace(*params.AllowedSubjects)
	}

	enableTokenReview := refs.BoolValue(params.EnableTokenReview)
	apiServerURL := normalizeAPIServerURL(params.KubernetesAPIServerURL)
	if err := validateTokenReviewConfig(enableTokenReview, apiServerURL); err != nil {
		log.Debug().Err(err).Msg("invalid token review config")
		return nil, nil, err
	}

	issuer, err := p.StorageProvider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		ClientID:        params.ServiceAccountID,
		Name:            strings.TrimSpace(params.Name),
		IssuerURL:       params.IssuerURL,
		KeySourceType:   params.KeySourceType,
		JWKSUrl:         params.JwksURL,
		ExpectedAud:     params.ExpectedAud,
		SubjectClaim:    subjectClaim,
		AllowedSubjects: allowedSubjects,
		IssuerType:      params.IssuerType,
		// Set explicitly rather than relying on the GORM column default so NoSQL
		// providers (no default support) persist the same values. This admin op
		// only ever creates client_assertion_trust rows; org-scoped SSO connections
		// are created through the dedicated OIDC-connection admin API.
		Kind:                     constants.TrustKindClientAssertion,
		AuthMethod:               constants.AuthMethodJWTAssertion,
		IsActive:                 true,
		SpiffeRefreshHintSeconds: refs.Int64Value(params.SpiffeRefreshHintSeconds),
		EnableTokenReview:        enableTokenReview,
		KubernetesAPIServerURL:   apiServerURL,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed AddTrustedIssuer")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditTrustedIssuerCreatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeTrustedIssuer,
		ResourceID:   issuer.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return issuer.AsAPITrustedIssuer(), nil, nil
}

// UpdateTrustedIssuer mutates only the fields present in params (load-then-
// mutate, so the storage Save does not blank untouched columns). Requires
// super-admin auth.
func (p *provider) UpdateTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.UpdateTrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateTrustedIssuer").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetTrustedIssuerByID")
		if storage.IsNotFound(err) {
			return nil, nil, NotFound("trusted issuer not found")
		}
		return nil, nil, err
	}

	if params.Name != nil {
		issuer.Name = strings.TrimSpace(*params.Name)
	}
	if params.JwksURL != nil {
		issuer.JWKSUrl = params.JwksURL
	}
	if params.ExpectedAud != nil {
		if strings.TrimSpace(*params.ExpectedAud) == "" {
			log.Debug().Msg("expected_aud cannot be empty")
			return nil, nil, InvalidArgument("expected_aud cannot be empty")
		}
		issuer.ExpectedAud = *params.ExpectedAud
	}
	if params.AllowedSubjects != nil {
		// Empty is permitted and meaningful: it reverts the row to DENY-ALL.
		issuer.AllowedSubjects = strings.TrimSpace(*params.AllowedSubjects)
	}
	if params.IsActive != nil {
		issuer.IsActive = *params.IsActive
	}
	if params.SpiffeRefreshHintSeconds != nil {
		issuer.SpiffeRefreshHintSeconds = *params.SpiffeRefreshHintSeconds
	}
	if params.EnableTokenReview != nil {
		issuer.EnableTokenReview = *params.EnableTokenReview
	}
	if params.KubernetesAPIServerURL != nil {
		issuer.KubernetesAPIServerURL = normalizeAPIServerURL(params.KubernetesAPIServerURL)
	}
	// Validate the merged result so enabling review without a (previously stored)
	// apiserver URL, or persisting a malformed URL, fails at write time.
	if err := validateTokenReviewConfig(issuer.EnableTokenReview, issuer.KubernetesAPIServerURL); err != nil {
		log.Debug().Err(err).Msg("invalid token review config")
		return nil, nil, err
	}

	updated, err := p.StorageProvider.UpdateTrustedIssuer(ctx, issuer)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateTrustedIssuer")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditTrustedIssuerUpdatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeTrustedIssuer,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return updated.AsAPITrustedIssuer(), nil, nil
}

// DeleteTrustedIssuer removes a trusted issuer by id. Requires super-admin auth.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.TrustedIssuerRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteTrustedIssuer").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("trusted issuer ID required")
		return nil, nil, InvalidArgument("trusted issuer ID required")
	}

	issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetTrustedIssuerByID")
		if storage.IsNotFound(err) {
			return nil, nil, NotFound("trusted issuer not found")
		}
		return nil, nil, err
	}

	if err := p.StorageProvider.DeleteTrustedIssuer(ctx, issuer); err != nil {
		log.Debug().Err(err).Msg("failed DeleteTrustedIssuer")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditTrustedIssuerDeletedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeTrustedIssuer,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: "Trusted issuer deleted successfully",
	}, nil, nil
}

// TrustedIssuer returns a single trusted issuer by id. Requires super-admin auth.
func (p *provider) TrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.TrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "TrustedIssuer").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetTrustedIssuerByID")
		if storage.IsNotFound(err) {
			return nil, nil, NotFound("trusted issuer not found")
		}
		return nil, nil, err
	}
	return issuer.AsAPITrustedIssuer(), nil, nil
}

// TrustedIssuers returns a paginated list of trusted issuers, optionally
// filtered by service_account_id. Requires super-admin auth.
func (p *provider) TrustedIssuers(ctx context.Context, meta RequestMetadata, params *model.ListTrustedIssuersRequest) (*model.TrustedIssuers, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "TrustedIssuers").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	var paginatedReq *model.PaginationRequest
	var serviceAccountID string
	if params != nil {
		paginatedReq = params.Pagination
		serviceAccountID = refs.StringValue(params.ServiceAccountID)
	}
	pagination := utils.GetPagination(paginatedReq)

	issuers, pagination, err := p.StorageProvider.ListTrustedIssuers(ctx, serviceAccountID, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListTrustedIssuers")
		return nil, nil, err
	}
	res := make([]*model.TrustedIssuer, len(issuers))
	for i, issuer := range issuers {
		res[i] = issuer.AsAPITrustedIssuer()
	}
	return &model.TrustedIssuers{
		Pagination:     pagination,
		TrustedIssuers: res,
	}, nil, nil
}
