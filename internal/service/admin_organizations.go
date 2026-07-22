package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// validateOrgSlug rejects characters that would break the organization name's
// role as a URL-safe slug and, defense-in-depth, as an FGA object-id component:
// '/' is the FGA namespace separator this codebase uses (e.g. group:<org>/<id>),
// ':' and '#' are FGA-reserved, and whitespace/control characters have no place
// in a slug. FGA namespacing keys off the org UUID, not the slug, so today this
// is belt-and-suspenders — but the cross-tenant containment argument depends on
// namespacing keys staying slash-free, so the invariant is enforced by
// construction here rather than left to assumption.
func validateOrgSlug(name string) error {
	for _, r := range name {
		if r == '/' || r == ':' || r == '#' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return InvalidArgument(fmt.Sprintf("organization name must be a URL-safe slug: the character %q is not allowed", string(r)))
		}
	}
	return nil
}

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
		return nil, nil, InvalidArgument("name is required")
	}
	if err := validateOrgSlug(name); err != nil {
		log.Debug().Err(err).Msg("invalid organization name")
		p.logOrgFailure(meta, constants.AuditOrganizationCreateFailedEvent, "")
		return nil, nil, err
	}

	// Unique-name pre-check. The storage layer also enforces uniqueness (unique
	// index / check-then-insert); this yields a clear error before the write.
	if existing, _ := p.StorageProvider.GetOrganizationByName(ctx, name); existing != nil {
		log.Debug().Msg("organization name already exists")
		p.logOrgFailure(meta, constants.AuditOrganizationCreateFailedEvent, "")
		return nil, nil, AlreadyExists("an organization with this name already exists")
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
			return nil, nil, InvalidArgument("name cannot be empty")
		}
		if err := validateOrgSlug(name); err != nil {
			log.Debug().Err(err).Msg("invalid organization name")
			p.logOrgFailure(meta, constants.AuditOrganizationUpdateFailedEvent, params.ID)
			return nil, nil, err
		}
		if name != org.Name {
			if existing, _ := p.StorageProvider.GetOrganizationByName(ctx, name); existing != nil {
				log.Debug().Msg("organization name already exists")
				p.logOrgFailure(meta, constants.AuditOrganizationUpdateFailedEvent, params.ID)
				return nil, nil, AlreadyExists("an organization with this name already exists")
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
		return nil, nil, InvalidArgument("organization ID required")
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

// Organization returns a single organization by id. Requires super-admin, or
// org-admin of this specific org.
func (p *provider) Organization(ctx context.Context, meta RequestMetadata, params *model.OrganizationRequest) (*model.Organization, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Organization").Logger()
	// Super-admin or an org-admin of this specific org — same gating as
	// OrgMembers/AddOrgMember, so an org-scoped admin can view their own
	// org's detail page (the dashboard's OrganizationDetail view) without
	// needing instance-wide super-admin rights. requireOrgAdmin rejects
	// access to any OTHER org.
	if err := p.requireOrgAdmin(ctx, meta, params.ID); err != nil {
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
// Gated on params.OrgID: super-admin or an org-admin of that org. An org-admin
// may grant constants.OrgRoleAdmin to another member of their own org
// (delegated administration, bounded to their org — design invariant 5).
func (p *provider) AddOrgMember(ctx context.Context, meta RequestMetadata, params *model.AddOrgMemberRequest) (*model.OrgMember, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddOrgMember").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}

	orgID := strings.TrimSpace(params.OrgID)
	userID := strings.TrimSpace(params.UserID)
	if orgID == "" || userID == "" {
		log.Debug().Msg("org_id and user_id are required")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, InvalidArgument("org_id and user_id are required")
	}

	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Msg("organization not found")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, NotFound("organization not found")
	}

	if _, err := p.StorageProvider.GetUserByID(ctx, userID); err != nil {
		log.Debug().Err(err).Msg("user not found")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, NotFound("user not found")
	}

	// Membership uniqueness pre-check. The storage layer also enforces it.
	if existing, _ := p.StorageProvider.GetOrgMembership(ctx, orgID, userID); existing != nil {
		log.Debug().Msg("user is already a member of this organization")
		p.logOrgMemberFailure(meta, orgID)
		return nil, nil, AlreadyExists("user is already a member of this organization")
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

// RemoveOrgMember removes a user from an organization. Gated on params.OrgID:
// super-admin or an org-admin of that org.
//
// Last-admin guard (design invariant 5): a NON-super-admin caller cannot remove
// the org's final constants.OrgRoleAdmin holder — that would lock the org out
// of self-service. A super-admin is exempt (can always recover the org).
func (p *provider) RemoveOrgMember(ctx context.Context, meta RequestMetadata, params *model.RemoveOrgMemberRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RemoveOrgMember").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}

	orgID := strings.TrimSpace(params.OrgID)
	userID := strings.TrimSpace(params.UserID)
	if orgID == "" || userID == "" {
		log.Debug().Msg("org_id and user_id are required")
		p.logOrgMemberRemoveFailure(meta, orgID)
		return nil, nil, InvalidArgument("org_id and user_id are required")
	}

	membership, err := p.StorageProvider.GetOrgMembership(ctx, orgID, userID)
	if err != nil {
		log.Debug().Err(err).Msg("membership not found")
		p.logOrgMemberRemoveFailure(meta, orgID)
		return nil, nil, NotFound("membership not found")
	}

	// Last-admin guard. Skip for super-admins (recovery escape hatch).
	if p.requireSuperAdmin(ctx, meta) != nil && orgMembershipHasRole(membership, constants.OrgRoleAdmin) {
		hasOther, err := p.orgHasAdminOtherThan(ctx, orgID, userID)
		if err != nil {
			// Fail closed: if we cannot confirm another admin exists, keep this one.
			log.Debug().Err(err).Msg("failed to verify remaining org admins")
			p.logOrgMemberRemoveFailure(meta, orgID)
			return nil, nil, fmt.Errorf("could not verify remaining organization admins")
		}
		if !hasOther {
			log.Debug().Msg("refusing to remove the last org admin")
			p.logOrgMemberRemoveFailure(meta, orgID)
			return nil, nil, FailedPrecondition(fmt.Sprintf("cannot remove the last %s of the organization", constants.OrgRoleAdmin))
		}
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

// OrgMembers returns a paginated list of an organization's members. Gated on
// params.OrgID: super-admin or an org-admin of that org (never another org's).
func (p *provider) OrgMembers(ctx context.Context, meta RequestMetadata, params *model.ListOrgMembersRequest) (*model.OrgMembers, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "OrgMembers").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}

	if params.OrgID == "" {
		log.Debug().Msg("org_id is required")
		return nil, nil, InvalidArgument("org_id is required")
	}
	pagination := utils.GetPagination(params.Pagination)

	memberships, pagination, err := p.StorageProvider.ListOrgMembershipsByOrg(ctx, params.OrgID, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListOrgMembershipsByOrg")
		return nil, nil, err
	}
	res := make([]*model.OrgMember, len(memberships))
	for i, m := range memberships {
		member := m.AsAPIOrgMember()
		// Resolve the member's user identity for display. One lookup per member
		// per page; acceptable since the list is paginated. A dangling user
		// reference must not break the listing, so leave the identity blank.
		if user, uErr := p.StorageProvider.GetUserByID(ctx, m.UserID); uErr == nil {
			member.Email = user.Email
			member.GivenName = user.GivenName
			member.FamilyName = user.FamilyName
		} else {
			log.Debug().Err(uErr).Str("user_id", m.UserID).Msg("failed GetUserByID; leaving member identity blank")
		}
		res[i] = member
	}
	return &model.OrgMembers{
		Pagination: pagination,
		OrgMembers: res,
	}, nil, nil
}

// UserOrganizations returns the organizations a user belongs to along with the
// roles held in each. Requires super-admin auth (same gating as other _user/
// _users admin ops). Backs the admin _user_organizations query, called lazily
// by the dashboard user detail view.
func (p *provider) UserOrganizations(ctx context.Context, meta RequestMetadata, params *model.UserOrganizationsRequest) (*model.UserOrganizations, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UserOrganizations").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if params == nil || strings.TrimSpace(params.UserID) == "" {
		log.Debug().Msg("user_id is required")
		return nil, nil, InvalidArgument("user_id is required")
	}

	var pagination *model.Pagination
	if params.Pagination != nil {
		pagination = utils.GetPagination(&model.PaginatedRequest{Pagination: params.Pagination})
	} else {
		pagination = utils.GetPagination(nil)
	}

	memberships, pagination, err := p.StorageProvider.ListOrgMembershipsByUser(ctx, params.UserID, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListOrgMembershipsByUser")
		return nil, nil, err
	}

	res := make([]*model.UserOrganization, 0, len(memberships))
	for _, m := range memberships {
		org, err := p.StorageProvider.GetOrganizationByID(ctx, m.OrgID)
		if err != nil {
			// A membership referencing a missing org is inconsistent but must
			// not break the whole listing; skip it.
			log.Debug().Err(err).Str("org_id", m.OrgID).Msg("failed GetOrganizationByID; skipping membership")
			continue
		}
		res = append(res, &model.UserOrganization{
			Organization: org.AsAPIOrganization(),
			Roles:        m.ParsedRoles(),
		})
	}

	return &model.UserOrganizations{
		Pagination:        pagination,
		UserOrganizations: res,
	}, nil, nil
}

// orgMembershipHasRole reports whether the membership carries role.
func orgMembershipHasRole(m *schemas.OrgMembership, role string) bool {
	for _, r := range m.ParsedRoles() {
		if r == role {
			return true
		}
	}
	return false
}

// orgHasAdminOtherThan reports whether orgID has at least one member holding
// constants.OrgRoleAdmin whose user id is not excludeUserID. It backs the
// last-admin guard, so it short-circuits as soon as a second admin is found and
// pages through all members otherwise.
//
// KNOWN LIMITATION (check-then-delete, non-atomic): two concurrent
// RemoveOrgMember calls each removing a *different* one of exactly two admins
// can both observe "another admin exists" and both proceed, leaving the org
// with zero admins. This is self-inflicted, single-tenant, and always
// recoverable by a super-admin (the designed recovery path), so it is accepted
// for v1. A fully atomic guard would need a storage-layer conditional delete /
// count-in-transaction, which 3 of the 6 NoSQL providers cannot express
// uniformly; not worth it for a contained, recoverable race.
func (p *provider) orgHasAdminOtherThan(ctx context.Context, orgID, excludeUserID string) (bool, error) {
	const pageLimit = int64(100)
	offset := int64(0)
	for {
		page := &model.Pagination{Limit: pageLimit, Offset: offset, Page: offset/pageLimit + 1}
		members, pg, err := p.StorageProvider.ListOrgMembershipsByOrg(ctx, orgID, page)
		if err != nil {
			return false, err
		}
		for _, m := range members {
			if m.UserID == excludeUserID {
				continue
			}
			if orgMembershipHasRole(m, constants.OrgRoleAdmin) {
				return true, nil
			}
		}
		offset += int64(len(members))
		if len(members) < int(pageLimit) || pg == nil || offset >= pg.Total {
			return false, nil
		}
	}
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
