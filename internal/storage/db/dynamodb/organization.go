package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrganization creates a new organization record. name is unique;
// DynamoDB has no cross-attribute unique constraint, so guard with a
// check-then-insert on the name GSI (closes the sequential case).
func (p *provider) AddOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	org.Key = org.ID
	now := time.Now().Unix()
	org.CreatedAt = now
	org.UpdatedAt = now
	if existing, _ := p.GetOrganizationByName(ctx, org.Name); existing != nil {
		return nil, fmt.Errorf("organization with %s name already exists", org.Name)
	}
	if err := p.putItem(ctx, schemas.Collections.Organization, org); err != nil {
		return nil, err
	}
	return org, nil
}

// UpdateOrganization updates an organization record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge, so a partial struct
// blanks untouched columns to their zero values.
func (p *provider) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	org.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Organization, "id", org.ID, org); err != nil {
		return nil, err
	}
	return org, nil
}

// DeleteOrganization removes an organization and all its memberships.
// Deletes child memberships BEFORE the parent (mirrors DeleteClient ordering);
// any query/delete error is returned so the caller knows the cascade did not
// complete.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	if org == nil {
		return nil
	}
	items, err := p.queryEq(ctx, schemas.Collections.OrgMembership, "org_id", "org_id", org.ID, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var membership schemas.OrgMembership
		if err := unmarshalItem(it, &membership); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.OrgMembership, "id", membership.ID); err != nil {
			return err
		}
	}
	// Cascade verified domains — otherwise the domain becomes permanently
	// unclaimable (it is the unique partition key of org_domains).
	if err := p.DeleteOrgDomainsByOrg(ctx, org.ID); err != nil {
		return err
	}
	return p.deleteItemByHash(ctx, schemas.Collections.Organization, "id", org.ID)
}

// GetOrganizationByID fetches an organization by primary key.
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	var org schemas.Organization
	err := p.getItemByHash(ctx, schemas.Collections.Organization, "id", id, &org)
	if err != nil {
		return nil, err
	}
	if org.ID == "" {
		return nil, errors.New("no document found")
	}
	return &org, nil
}

// GetOrganizationByName fetches an organization by its unique name slug.
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.Organization, "name", "name", name, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var org schemas.Organization
	if err := unmarshalItem(items[0], &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// ListOrganizations returns a paginated list of organizations.
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	paginationClone := pagination
	var orgs []*schemas.Organization

	items, err := p.scanAllRaw(ctx, schemas.Collections.Organization, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, it := range items {
		var org schemas.Organization
		if err := unmarshalItem(it, &org); err != nil {
			return nil, nil, err
		}
		orgs = append(orgs, &org)
	}

	sort.Slice(orgs, func(i, j int) bool { return orgs[i].CreatedAt > orgs[j].CreatedAt })
	paginationClone.Total = int64(len(orgs))

	start := int(pagination.Offset)
	if start >= len(orgs) {
		return []*schemas.Organization{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(orgs) {
		end = len(orgs)
	}
	return orgs[start:end], paginationClone, nil
}
