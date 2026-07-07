package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// defaultSubjectClaim is the JWT claim used to identify the workload when the
// admin does not specify one (spec §6 AddTrustedIssuerRequest comment).
const defaultSubjectClaim = "sub"

// AddTrustedIssuer registers an external JWT issuer for a service account.
// subject_claim defaults to "sub" when omitted. Requires super-admin auth.
func (p *provider) AddTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.AddTrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddTrustedIssuer").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(params.ServiceAccountID) == "" {
		return nil, nil, fmt.Errorf("service_account_id is required")
	}
	if strings.TrimSpace(params.Name) == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(params.IssuerURL) == "" {
		return nil, nil, fmt.Errorf("issuer_url is required")
	}
	if strings.TrimSpace(params.KeySourceType) == "" {
		return nil, nil, fmt.Errorf("key_source_type is required")
	}
	if strings.TrimSpace(params.ExpectedAud) == "" {
		return nil, nil, fmt.Errorf("expected_aud is required")
	}
	if strings.TrimSpace(params.IssuerType) == "" {
		return nil, nil, fmt.Errorf("issuer_type is required")
	}

	// Reject issuers bound to a non-existent service account — otherwise a typo
	// creates an orphan that can never be reached via the parent.
	if _, err := p.StorageProvider.GetServiceAccountByID(ctx, params.ServiceAccountID); err != nil {
		log.Debug().Err(err).Str("service_account_id", params.ServiceAccountID).Msg("service account not found")
		return nil, nil, fmt.Errorf("service account not found: %s", params.ServiceAccountID)
	}

	// Enforce issuer_url uniqueness at the service layer. Storage providers use a
	// plain insert with no unique constraint, so duplicates would otherwise
	// coexist and GetTrustedIssuerByIssuerURL (called on every client_assertion
	// validation) would resolve nondeterministically. This guard protects all
	// backends uniformly.
	if existing, err := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, params.IssuerURL); err == nil && existing != nil {
		log.Debug().Str("issuer_url", params.IssuerURL).Msg("issuer_url already registered")
		return nil, nil, fmt.Errorf("issuer_url already registered: %s", params.IssuerURL)
	}

	subjectClaim := defaultSubjectClaim
	if params.SubjectClaim != nil && strings.TrimSpace(*params.SubjectClaim) != "" {
		subjectClaim = strings.TrimSpace(*params.SubjectClaim)
	}

	issuer, err := p.StorageProvider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		ServiceAccountID: params.ServiceAccountID,
		Name:             strings.TrimSpace(params.Name),
		IssuerURL:        params.IssuerURL,
		KeySourceType:    params.KeySourceType,
		JWKSUrl:          params.JwksURL,
		ExpectedAud:      params.ExpectedAud,
		SubjectClaim:     subjectClaim,
		IssuerType:       params.IssuerType,
		// Set explicitly rather than relying on the GORM column default so NoSQL
		// providers (no default support) persist the same values.
		AuthMethod:               "jwt_assertion",
		IsActive:                 true,
		SpiffeRefreshHintSeconds: refs.Int64Value(params.SpiffeRefreshHintSeconds),
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
			return nil, nil, fmt.Errorf("expected_aud cannot be empty")
		}
		issuer.ExpectedAud = *params.ExpectedAud
	}
	if params.IsActive != nil {
		issuer.IsActive = *params.IsActive
	}
	if params.SpiffeRefreshHintSeconds != nil {
		issuer.SpiffeRefreshHintSeconds = *params.SpiffeRefreshHintSeconds
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
		return nil, nil, fmt.Errorf("trusted issuer ID required")
	}

	issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetTrustedIssuerByID")
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

	var paginatedReq *model.PaginatedRequest
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
