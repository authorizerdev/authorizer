package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrganization to save a organization in database
func (p *provider) AddOrganization(ctx context.Context, v *schemas.Organization) (*schemas.Organization, error) {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	v.Key = v.ID
	now := time.Now().Unix()
	v.CreatedAt = now
	v.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.Organization, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateOrganization to update a organization in database. Callers MUST load the record first:
// every field is written.
func (p *provider) UpdateOrganization(ctx context.Context, v *schemas.Organization) (*schemas.Organization, error) {
	if v.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	v.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.Organization, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DeleteOrganization deletes an organization with its memberships and
// verified domains, atomically. A domain left behind would stay claimed (it
// is the unique id of org_domains) and could never be verified again.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	return p.transaction(ctx, `DELETE type::table($mtb) WHERE org_id = $id RETURN NONE;
		DELETE type::table($dtb) WHERE org_id = $id RETURN NONE;
		DELETE type::record($tb, $id) RETURN NONE;`, map[string]any{
		"mtb": schemas.Collections.OrgMembership, "dtb": schemas.Collections.OrgDomain,
		"tb": schemas.Collections.Organization, "id": org.ID,
	})
}

// GetOrganizationByID to get an organization by ID
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	return getByID[schemas.Organization](ctx, p, schemas.Collections.Organization, id, "organization")
}

// GetOrganizationByName to get an organization by name
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	return getWhere[schemas.Organization](ctx, p, schemas.Collections.Organization, "name = $v", "organization", map[string]any{"v": name})
}

// ListOrganizations to list organizations, newest first
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	rows, pag, err := page[schemas.Organization](ctx, p, schemas.Collections.Organization, "", nil, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}
