package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const organizationColumns = "_id, name, display_name, enabled, created_at, updated_at"

// AddOrganization creates a new organization record. name is unique;
// Couchbase has no cross-attribute unique constraint, so guard with a
// check-then-insert on name (closes the sequential case).
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(org)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.Organization).Insert(org.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// UpdateOrganization updates an organization record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	org.UpdatedAt = time.Now().Unix()
	orgMap, err := structToDocument(org)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(orgMap)
	params["_id"] = org.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Organization, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return org, nil
}

// DeleteOrganization removes an organization and all its memberships.
// Mirrors the DeleteClient cascade-delete pattern.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Organization).Remove(org.ID, &removeOpt)
	if err != nil {
		return err
	}
	params := make(map[string]interface{}, 1)
	params["org_id"] = org.ID
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE org_id=$org_id`, p.scopeName, schemas.Collections.OrgMembership)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	// Cascade verified domains — otherwise the domain becomes permanently
	// unclaimable (it is the unique document key of org_domains).
	return p.DeleteOrgDomainsByOrg(ctx, org.ID)
}

// GetOrganizationByID fetches an organization by primary key.
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, organizationColumns, p.scopeName, schemas.Collections.Organization)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	org := &schemas.Organization{}
	if err := decodeDocument(raw, org); err != nil {
		return nil, err
	}
	return org, nil
}

// GetOrganizationByName fetches an organization by its unique name slug.
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	params := make(map[string]interface{}, 1)
	params["name"] = name
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE name=$name LIMIT 1`, organizationColumns, p.scopeName, schemas.Collections.Organization)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	org := &schemas.Organization{}
	if err := decodeDocument(raw, org); err != nil {
		return nil, err
	}
	return org, nil
}

// ListOrganizations returns a paginated list of organizations.
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	orgs := []*schemas.Organization{}
	paginationClone := pagination
	params := make(map[string]interface{}, 2)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Organization)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", organizationColumns, p.scopeName, schemas.Collections.Organization)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, nil, err
		}
		org := &schemas.Organization{}
		if err := decodeDocument(raw, org); err != nil {
			return nil, nil, err
		}
		orgs = append(orgs, org)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return orgs, paginationClone, nil
}
