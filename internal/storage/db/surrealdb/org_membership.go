package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgMembership to save a organization membership in database
func (p *provider) AddOrgMembership(ctx context.Context, v *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	v.Key = v.ID
	now := time.Now().Unix()
	v.CreatedAt = now
	v.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.OrgMembership, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateOrgMembership to update a organization membership in database. Callers MUST load the record first:
// every field is written.
func (p *provider) UpdateOrgMembership(ctx context.Context, v *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if v.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	v.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.OrgMembership, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DeleteOrgMembership to delete an organization membership
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	return p.deleteByID(ctx, schemas.Collections.OrgMembership, membership.ID)
}

// GetOrgMembership to get a user's membership of an organization
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	return getWhere[schemas.OrgMembership](ctx, p, schemas.Collections.OrgMembership, "org_id = $o AND user_id = $u", "organization membership", map[string]any{"o": orgID, "u": userID})
}

// ListOrgMembershipsByOrg to list an organization's memberships, newest first
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	rows, pag, err := page[schemas.OrgMembership](ctx, p, schemas.Collections.OrgMembership, "org_id = $o", map[string]any{"o": orgID}, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// ListOrgMembershipsByUser to list a user's memberships, newest first
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	rows, pag, err := page[schemas.OrgMembership](ctx, p, schemas.Collections.OrgMembership, "user_id = $u", map[string]any{"u": userID}, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}
