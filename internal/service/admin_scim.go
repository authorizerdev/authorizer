package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service/scim"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// CreateScimEndpoint provisions the per-org inbound SCIM connection and returns
// its bearer token exactly once (bcrypt-hashed at rest, crypto/rand entropy).
// One endpoint per org. Gated on params.OrgID: super-admin or that org's
// org-admin (see constants.OrgRoleAdmin).
func (p *provider) CreateScimEndpoint(ctx context.Context, meta RequestMetadata, params *model.CreateScimEndpointRequest) (*model.CreateScimEndpointResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateScimEndpoint").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	if orgID == "" {
		p.logScimFailure(meta, constants.AuditScimEndpointCreateFailedEvent, "")
		return nil, nil, fmt.Errorf("org_id is required")
	}
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Msg("organization not found")
		p.logScimFailure(meta, constants.AuditScimEndpointCreateFailedEvent, orgID)
		return nil, nil, fmt.Errorf("organization not found")
	}
	if existing, _ := p.StorageProvider.GetScimEndpointByOrgID(ctx, orgID); existing != nil {
		log.Debug().Msg("scim endpoint already exists for org")
		p.logScimFailure(meta, constants.AuditScimEndpointCreateFailedEvent, orgID)
		return nil, nil, fmt.Errorf("a scim endpoint already exists for this organization")
	}

	// The endpoint id is embedded (non-secret) in the token; generate it first
	// so the token references the row it will authenticate.
	id := uuid.New().String()
	token, hash, err := scim.GenerateToken(id)
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate scim token")
		p.logScimFailure(meta, constants.AuditScimEndpointCreateFailedEvent, orgID)
		return nil, nil, err
	}
	endpoint, err := p.StorageProvider.AddScimEndpoint(ctx, &schemas.ScimEndpoint{
		ID:        id,
		OrgID:     orgID,
		TokenHash: hash,
		// Set Enabled explicitly — never rely on the GORM default:true quirk.
		Enabled: true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to add scim endpoint")
		p.logScimFailure(meta, constants.AuditScimEndpointCreateFailedEvent, orgID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditScimEndpointCreatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeScimEndpoint,
		ResourceID:   endpoint.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.CreateScimEndpointResponse{
		ScimEndpoint: endpoint.AsAPIScimEndpoint(),
		Token:        token,
	}, nil, nil
}

// RotateScimToken mints a fresh bearer token for an org's endpoint, invalidating
// the previous one, and returns it once. The endpoint is keyed solely by
// org_id, so gating on params.OrgID (super-admin or that org's org-admin) is the
// resource's real OrgID — no id-vs-org confused-deputy vector here.
func (p *provider) RotateScimToken(ctx context.Context, meta RequestMetadata, params *model.ScimEndpointRequest) (*model.CreateScimEndpointResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RotateScimToken").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	endpoint, err := p.StorageProvider.GetScimEndpointByOrgID(ctx, orgID)
	if err != nil || endpoint == nil {
		log.Debug().Err(err).Msg("scim endpoint not found")
		p.logScimFailure(meta, constants.AuditScimTokenRotateFailedEvent, orgID)
		return nil, nil, fmt.Errorf("scim endpoint not found")
	}
	token, hash, err := scim.GenerateToken(endpoint.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate scim token")
		p.logScimFailure(meta, constants.AuditScimTokenRotateFailedEvent, orgID)
		return nil, nil, err
	}
	endpoint.TokenHash = hash
	updated, err := p.StorageProvider.UpdateScimEndpoint(ctx, endpoint)
	if err != nil {
		log.Debug().Err(err).Msg("failed to update scim endpoint")
		p.logScimFailure(meta, constants.AuditScimTokenRotateFailedEvent, orgID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditScimTokenRotatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeScimEndpoint,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.CreateScimEndpointResponse{
		ScimEndpoint: updated.AsAPIScimEndpoint(),
		Token:        token,
	}, nil, nil
}

// DeleteScimEndpoint removes an org's SCIM connection. Gated on params.OrgID
// (super-admin or that org's org-admin); org_id is the resource's sole key.
func (p *provider) DeleteScimEndpoint(ctx context.Context, meta RequestMetadata, params *model.ScimEndpointRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteScimEndpoint").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	endpoint, err := p.StorageProvider.GetScimEndpointByOrgID(ctx, orgID)
	if err != nil || endpoint == nil {
		log.Debug().Err(err).Msg("scim endpoint not found")
		p.logScimFailure(meta, constants.AuditScimEndpointDeleteFailedEvent, orgID)
		return nil, nil, fmt.Errorf("scim endpoint not found")
	}
	if err := p.StorageProvider.DeleteScimEndpoint(ctx, endpoint); err != nil {
		log.Debug().Err(err).Msg("failed to delete scim endpoint")
		p.logScimFailure(meta, constants.AuditScimEndpointDeleteFailedEvent, orgID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditScimEndpointDeletedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeScimEndpoint,
		ResourceID:   endpoint.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "scim endpoint deleted successfully"}, nil, nil
}

// ScimEndpoint returns an org's SCIM endpoint metadata (never the token).
// Gated on params.OrgID (super-admin or that org's org-admin).
func (p *provider) ScimEndpoint(ctx context.Context, meta RequestMetadata, params *model.ScimEndpointRequest) (*model.ScimEndpoint, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ScimEndpoint").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	endpoint, err := p.StorageProvider.GetScimEndpointByOrgID(ctx, strings.TrimSpace(params.OrgID))
	if err != nil || endpoint == nil {
		log.Debug().Err(err).Msg("scim endpoint not found")
		return nil, nil, fmt.Errorf("scim endpoint not found")
	}
	return endpoint.AsAPIScimEndpoint(), nil, nil
}

// logScimFailure records a failed SCIM endpoint admin operation.
func (p *provider) logScimFailure(meta RequestMetadata, action, orgID string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:   action,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeScimEndpoint,
		ResourceID:   orgID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
}
