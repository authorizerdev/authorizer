package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgDomain atomically inserts a verified domain row. The document _key is
// the normalized domain, so a duplicate CreateDocument is rejected by ArangoDB
// (unique-constraint conflict) — first-writer wins with no check-then-insert
// race. On any create error we classify by re-reading the row: same org →
// idempotent success, different org → ErrOrgDomainConflict.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	orgDomainCollection, err := p.db.Collection(ctx, schemas.Collections.OrgDomain)
	if err != nil {
		return nil, err
	}
	doc, err := structToDocument(domain)
	if err != nil {
		return nil, err
	}
	if _, err := orgDomainCollection.CreateDocument(ctx, doc); err != nil {
		existing, getErr := p.GetOrgDomainByDomain(ctx, domain.ID)
		if getErr == nil && existing != nil {
			if existing.OrgID == domain.OrgID {
				return existing, nil
			}
			return nil, schemas.ErrOrgDomainConflict
		}
		return nil, err
	}
	return domain, nil
}

// GetOrgDomainByDomain fetches a verified domain by its normalized value.
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	var orgDomain *schemas.OrgDomain
	query := fmt.Sprintf("FOR d in %s FILTER d.domain == @domain LIMIT 1 RETURN d", schemas.Collections.OrgDomain)
	bindVars := map[string]interface{}{
		"domain": domain,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if orgDomain == nil {
				return nil, fmt.Errorf("org domain not found")
			}
			break
		}
		d := &schemas.OrgDomain{}
		if _, err := readDocument(ctx, cursor, d); err != nil {
			return nil, err
		}
		orgDomain = d
	}
	return orgDomain, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	domains := []*schemas.OrgDomain{}
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.OrgDomain, pagination.Offset, pagination.Limit)
	bindVars := map[string]interface{}{
		"org_id": orgID,
	}
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, bindVars)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		d := &schemas.OrgDomain{}
		meta, err := readDocument(ctx, cursor, d)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			domains = append(domains, d)
		}
	}
	return domains, paginationClone, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
// The _key is the domain, so remove by key.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	orgDomainCollection, err := p.db.Collection(ctx, schemas.Collections.OrgDomain)
	if err != nil {
		return err
	}
	if _, err := orgDomainCollection.RemoveDocument(ctx, domain); err != nil {
		if arangoDriver.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.org_id == @org_id REMOVE { _key: d._key } IN %s", schemas.Collections.OrgDomain, schemas.Collections.OrgDomain)
	bindVars := map[string]interface{}{
		"org_id": orgID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}
