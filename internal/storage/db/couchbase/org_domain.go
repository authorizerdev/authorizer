package couchbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const orgDomainColumns = "_id, org_id, domain, verified_at, created_at, updated_at"

// AddOrgDomain atomically inserts a verified domain row. The document key is the
// normalized domain and the write is an Insert (not Upsert), so a duplicate is
// rejected by Couchbase (ErrDocumentExists) — unlike the scim_endpoint
// check-then-insert guard, there is NO TOCTOU race. On a lost race we classify
// the existing row by owning org.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	doc, err := structToDocument(domain)
	if err != nil {
		return nil, err
	}
	insertOpt := gocb.InsertOptions{Context: ctx}
	_, err = p.db.Collection(schemas.Collections.OrgDomain).Insert(domain.ID, doc, &insertOpt)
	if err != nil {
		if errors.Is(err, gocb.ErrDocumentExists) {
			existing, getErr := p.GetOrgDomainByDomain(ctx, domain.ID)
			if getErr == nil && existing != nil {
				if existing.OrgID == domain.OrgID {
					return existing, nil
				}
				return nil, schemas.ErrOrgDomainConflict
			}
		}
		return nil, err
	}
	return domain, nil
}

// GetOrgDomainByDomain fetches a verified domain by its normalized value.
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	params := map[string]interface{}{"_id": domain}
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, orgDomainColumns, p.scopeName, schemas.Collections.OrgDomain)
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
	d := &schemas.OrgDomain{}
	if err := decodeDocument(raw, d); err != nil {
		return nil, err
	}
	return d, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	domains := []*schemas.OrgDomain{}
	paginationClone := pagination
	table := fmt.Sprintf("%s.%s", p.scopeName, schemas.Collections.OrgDomain)

	params := map[string]interface{}{
		"offset":       paginationClone.Offset,
		"limit":        paginationClone.Limit,
		"filter_value": orgID,
	}
	total := TotalDocs{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s WHERE org_id=$filter_value", table)
	countRes, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: map[string]interface{}{"filter_value": orgID},
	})
	if err != nil {
		return nil, nil, err
	}
	_ = countRes.One(&total)
	paginationClone.Total = total.Total

	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id=$filter_value ORDER BY created_at DESC OFFSET $offset LIMIT $limit", orgDomainColumns, table)
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
		d := &schemas.OrgDomain{}
		if err := decodeDocument(raw, d); err != nil {
			return nil, nil, err
		}
		domains = append(domains, d)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return domains, paginationClone, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	removeOpt := gocb.RemoveOptions{Context: ctx}
	_, err := p.db.Collection(schemas.Collections.OrgDomain).Remove(domain, &removeOpt)
	if err != nil && !errors.Is(err, gocb.ErrDocumentNotFound) {
		return err
	}
	return nil
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	params := map[string]interface{}{"org_id": orgID}
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE org_id=$org_id`, p.scopeName, schemas.Collections.OrgDomain)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	return err
}
