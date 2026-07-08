package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CreateOrganization provisions a new organization. The name must be a
// unique, non-empty slug. Requires super-admin auth.
func (p *provider) CreateOrganization(ctx context.Context, meta RequestMetadata, params *model.CreateOrganizationRequest) (*model.Organization, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateOrganization").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		log.Debug().Msg("name is required")
		p.logOrgFailure(meta, constants.AuditOrganizationCreateFailedEvent, "")
		return nil, nil, fmt.Errorf("name is required")
	}

	// Unique-name pre-check. The storage layer also enforces uniqueness (unique
	// index / check-then-insert); this yields a clear error before the write.
	if existing, _ := p.StorageProvider.GetOrganizationByName(ctx, name); existing != nil {
		log.Debug().Msg("organization name already exists")
		p.logOrgFailure(meta, constants.AuditOrganizationCreateFailedEvent, "")
		return nil, nil, fmt.Errorf("an organization with this name already exists")
	}

	org, err := p.StorageProvider.AddOrganization(ctx, &schemas.Organization{
		Name:        name,
		DisplayName: params.DisplayName,
		// Set Enabled explicitly — never rely on the GORM `default:true` column
		// default (a future create-as-disabled path would silently come back on).
		Enabled: true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to add organization")
		p.logOrgFailure(meta, constants.AuditOrganizationCreateFailedEvent, "")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrganizationCreatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrganization,
		ResourceID:   org.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return org.AsAPIOrganization(), nil, nil
}

// UpdateOrganization mutates only the fields present in params (load-then-
// mutate, so the storage Save does not blank untouched columns). A changed
// name is re-checked for uniqueness. Requires super-admin auth.
func (p *provider) UpdateOrganization(ctx context.Context, meta RequestMetadata, params *model.UpdateOrganizationRequest) (*model.Organization, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateOrganization").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	org, err := p.StorageProvider.GetOrganizationByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetOrganizationByID")
		p.logOrgFailure(meta, constants.AuditOrganizationUpdateFailedEvent, params.ID)
		return nil, nil, err
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			log.Debug().Msg("name cannot be empty")
			p.logOrgFailure(meta, constants.AuditOrganizationUpdateFailedEvent, params.ID)
			return nil, nil, fmt.Errorf("name cannot be empty")
		}
		if name != org.Name {
			if existing, _ := p.StorageProvider.GetOrganizationByName(ctx, name); existing != nil {
				log.Debug().Msg("organization name already exists")
				p.logOrgFailure(meta, constants.AuditOrganizationUpdateFailedEvent, params.ID)
				return nil, nil, fmt.Errorf("an organization with this name already exists")
			}
			org.Name = name
		}
	}
	if params.DisplayName != nil {
		org.DisplayName = params.DisplayName
	}
	if params.Enabled != nil {
		org.Enabled = *params.Enabled
	}

	updated, err := p.StorageProvider.UpdateOrganization(ctx, org)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateOrganization")
		p.logOrgFailure(meta, constants.AuditOrganizationUpdateFailedEvent, params.ID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrganizationUpdatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrganization,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return updated.AsAPIOrganization(), nil, nil
}

// DeleteOrganization removes an organization. The storage layer cascades to
// its memberships. Requires super-admin auth.
func (p *provider) DeleteOrganization(ctx context.Context, meta RequestMetadata, params *model.OrganizationRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteOrganization").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("organization ID required")
		p.logOrgFailure(meta, constants.AuditOrganizationDeleteFailedEvent, "")
		return nil, nil, fmt.Errorf("organization ID required")
	}

	org, err := p.StorageProvider.GetOrganizationByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetOrganizationByID")
		p.logOrgFailure(meta, constants.AuditOrganizationDeleteFailedEvent, params.ID)
		return nil, nil, err
	}

	if err := p.StorageProvider.DeleteOrganization(ctx, org); err != nil {
		log.Debug().Err(err).Msg("failed DeleteOrganization")
		p.logOrgFailure(meta, constants.AuditOrganizationDeleteFailedEvent, params.ID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrganizationDeletedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrganization,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: "Organization deleted successfully",
	}, nil, nil
}

// Organization returns a single organization by id. Requires super-admin auth.
func (p *provider) Organization(ctx context.Context, meta RequestMetadata, params *model.OrganizationRequest) (*model.Organization, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Organization").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	org, err := p.StorageProvider.GetOrganizationByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetOrganizationByID")
		return nil, nil, err
	}
	return org.AsAPIOrganization(), nil, nil
}

// Organizations returns a paginated list of organizations. Requires
// super-admin auth.
func (p *provider) Organizations(ctx context.Context, meta RequestMetadata, params *model.ListOrganizationsRequest) (*model.Organizations, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Organizations").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	var paginatedReq *model.PaginatedRequest
	if params != nil {
		paginatedReq = params.Pagination
	}
	pagination := utils.GetPagination(paginatedReq)

	orgs, pagination, err := p.StorageProvider.ListOrganizations(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListOrganizations")
		return nil, nil, err
	}
	res := make([]*model.Organization, len(orgs))
	for i, org := range orgs {
		res[i] = org.AsAPIOrganization()
	}
	return &model.Organizations{
		Pagination:    pagination,
		Organizations: res,
	}, nil, nil
}

// AddOrgMember adds a user to an organization with a set of per-org roles.
// The organization and user must exist and the (org, user) pair must be unique.
// Requires super-admin auth.
func (p *provider) AddOrgMember(ctx context.Context, meta RequestMetadata, params *model.AddOrgMemberRequest) (*model.OrgMember, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddOrgMember").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	orgID := strings.TrimSpace(params.OrgID)
	userID := strings.TrimSpace(params.UserID)
	if orgID == "" || userID == "" {
		log.Debug().Msg("org_id and user_id are required")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, fmt.Errorf("org_id and user_id are required")
	}

	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Msg("organization not found")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, fmt.Errorf("organization not found")
	}

	if _, err := p.StorageProvider.GetUserByID(ctx, userID); err != nil {
		log.Debug().Err(err).Msg("user not found")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, fmt.Errorf("user not found")
	}

	// Membership uniqueness pre-check. The storage layer also enforces it.
	if existing, _ := p.StorageProvider.GetOrgMembership(ctx, orgID, userID); existing != nil {
		log.Debug().Msg("user is already a member of this organization")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, fmt.Errorf("user is already a member of this organization")
	}

	membership, err := p.StorageProvider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  orgID,
		UserID: userID,
		Roles:  normalizeScopes(params.Roles),
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed AddOrgMembership")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrgMemberAddedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgMembership,
		ResourceID:   membership.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return membership.AsAPIOrgMember(), nil, nil
}

// RemoveOrgMember removes a user from an organization. Requires super-admin auth.
func (p *provider) RemoveOrgMember(ctx context.Context, meta RequestMetadata, params *model.RemoveOrgMemberRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RemoveOrgMember").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	orgID := strings.TrimSpace(params.OrgID)
	userID := strings.TrimSpace(params.UserID)
	if orgID == "" || userID == "" {
		log.Debug().Msg("org_id and user_id are required")
		p.logOrgMemberRemoveFailure(meta, orgID)
		return nil, nil, fmt.Errorf("org_id and user_id are required")
	}

	membership, err := p.StorageProvider.GetOrgMembership(ctx, orgID, userID)
	if err != nil {
		log.Debug().Err(err).Msg("membership not found")
		p.logOrgMemberRemoveFailure(meta, orgID)
		return nil, nil, fmt.Errorf("membership not found")
	}

	if err := p.StorageProvider.DeleteOrgMembership(ctx, membership); err != nil {
		log.Debug().Err(err).Msg("failed DeleteOrgMembership")
		p.logOrgMemberRemoveFailure(meta, orgID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrgMemberRemovedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgMembership,
		ResourceID:   membership.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: "Organization member removed successfully",
	}, nil, nil
}

// OrgMembers returns a paginated list of an organization's members. Requires
// super-admin auth.
func (p *provider) OrgMembers(ctx context.Context, meta RequestMetadata, params *model.ListOrgMembersRequest) (*model.OrgMembers, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "OrgMembers").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.OrgID == "" {
		log.Debug().Msg("org_id is required")
		return nil, nil, fmt.Errorf("org_id is required")
	}
	pagination := utils.GetPagination(params.Pagination)

	memberships, pagination, err := p.StorageProvider.ListOrgMembershipsByOrg(ctx, params.OrgID, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListOrgMembershipsByOrg")
		return nil, nil, err
	}
	res := make([]*model.OrgMember, len(memberships))
	for i, m := range memberships {
		res[i] = m.AsAPIOrgMember()
	}
	return &model.OrgMembers{
		Pagination: pagination,
		OrgMembers: res,
	}, nil, nil
}

// logOrgFailure records a failed organization admin operation.
func (p *provider) logOrgFailure(meta RequestMetadata, action, orgID string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:   action,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrganization,
		ResourceID:   orgID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
}

// logOrgMemberFailure records a failed add-member operation.
func (p *provider) logOrgMemberFailure(meta RequestMetadata, orgID string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrgMemberAddFailedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgMembership,
		ResourceID:   orgID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
}

// logOrgMemberRemoveFailure records a failed remove-member operation.
func (p *provider) logOrgMemberRemoveFailure(meta RequestMetadata, orgID string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOrgMemberRemoveFailedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgMembership,
		ResourceID:   orgID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
}
