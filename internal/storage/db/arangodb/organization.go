package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrganization creates a new organization record.
func (p *provider) AddOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	org.Key = org.ID
	now := time.Now().Unix()
	org.CreatedAt = now
	org.UpdatedAt = now
	orgCollection, _ := p.db.Collection(ctx, schemas.Collections.Organization)
	doc, err := structToDocument(org)
	if err != nil {
		return nil, err
	}
	meta, err := orgCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	org.Key = meta.Key
	org.ID = meta.ID.String()
	return org, nil
}

// UpdateOrganization updates an organization record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct.
func (p *provider) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	org.UpdatedAt = time.Now().Unix()
	orgCollection, _ := p.db.Collection(ctx, schemas.Collections.Organization)
	doc, err := structToDocument(org)
	if err != nil {
		return nil, err
	}
	meta, err := orgCollection.UpdateDocument(ctx, org.Key, doc)
	if err != nil {
		return nil, err
	}
	org.Key = meta.Key
	org.ID = meta.ID.String()
	return org, nil
}

// DeleteOrganization removes an organization and all its memberships.
// Mirrors the DeleteClient cascade-delete pattern.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	orgCollection, _ := p.db.Collection(ctx, schemas.Collections.Organization)
	_, err := orgCollection.RemoveDocument(ctx, org.Key)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("FOR d IN %s FILTER d.org_id == @org_id REMOVE { _key: d._key } IN %s", schemas.Collections.OrgMembership, schemas.Collections.OrgMembership)
	bindVars := map[string]interface{}{
		// OrgMembership.OrgID is stored as the bare key (set verbatim from the
		// external, API-facing id) — compare against org.Key, not org.ID (which
		// is the full "collection/key" handle after a read).
		"org_id": org.Key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}

// GetOrganizationByID fetches an organization by primary key.
// Filters on _key, not _id: every real caller holds the bare id
// AsAPIOrganization exposes, never the full "collection/key" handle.
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	var org *schemas.Organization
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.Organization)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if org == nil {
				return nil, fmt.Errorf("organization not found")
			}
			break
		}
		o := &schemas.Organization{}
		if _, err := readDocument(ctx, cursor, o); err != nil {
			return nil, err
		}
		org = o
	}
	return org, nil
}

// GetOrganizationByName fetches an organization by its unique name slug.
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	var org *schemas.Organization
	query := fmt.Sprintf("FOR d in %s FILTER d.name == @name LIMIT 1 RETURN d", schemas.Collections.Organization)
	bindVars := map[string]interface{}{
		"name": name,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if org == nil {
				return nil, fmt.Errorf("organization not found")
			}
			break
		}
		o := &schemas.Organization{}
		if _, err := readDocument(ctx, cursor, o); err != nil {
			return nil, err
		}
		org = o
	}
	return org, nil
}

// ListOrganizations returns a paginated list of organizations.
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	orgs := []*schemas.Organization{}
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Organization, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		org := &schemas.Organization{}
		meta, err := readDocument(ctx, cursor, org)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			orgs = append(orgs, org)
		}
	}
	return orgs, paginationClone, nil
}
