package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const orgDomainColumns = "id, org_id, domain, verified_at, created_at, updated_at"

// scanOrgDomain maps the orgDomainColumns projection onto a struct.
func scanOrgDomain(scan func(...interface{}) error, d *schemas.OrgDomain) error {
	return scan(&d.ID, &d.OrgID, &d.Domain, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt)
}

// AddOrgDomain atomically inserts a verified domain row. The normalized domain
// is the partition key and the insert is a lightweight transaction
// (INSERT ... IF NOT EXISTS), so first-writer-wins is enforced atomically —
// unlike the scim_endpoint check-then-insert guard, there is NO TOCTOU race. On
// a lost race the CAS returns the existing row, which we classify by owning org.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS", KeySpace+"."+schemas.Collections.OrgDomain, orgDomainColumns)
	existing := map[string]interface{}{}
	applied, err := p.db.Query(insertQuery, domain.ID, domain.OrgID, domain.Domain, domain.VerifiedAt, domain.CreatedAt, domain.UpdatedAt).MapScanCAS(existing)
	if err != nil {
		return nil, err
	}
	if !applied {
		// A row already exists for this domain; classify by owning org.
		if ownerOrg, ok := existing["org_id"].(string); ok && ownerOrg == domain.OrgID {
			return p.GetOrgDomainByDomain(ctx, domain.ID)
		}
		return nil, schemas.ErrOrgDomainConflict
	}
	return domain, nil
}

// GetOrgDomainByDomain fetches a verified domain by its normalized value (PK).
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	var d schemas.OrgDomain
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", orgDomainColumns, KeySpace+"."+schemas.Collections.OrgDomain)
	if err := scanOrgDomain(p.db.Query(query, domain).Consistency(gocql.One).Scan, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	domains := []*schemas.OrgDomain{}
	paginationClone := pagination
	table := KeySpace + "." + schemas.Collections.OrgDomain

	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE org_id = ? ALLOW FILTERING", table)
	if err := p.db.Query(totalCountQuery, orgID).Consistency(gocql.One).Scan(&paginationClone.Total); err != nil {
		return nil, nil, err
	}

	// there is no offset in cassandra: fetch limit + offset, return offset..limit
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? LIMIT %d ALLOW FILTERING", orgDomainColumns, table, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query, orgID).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var d schemas.OrgDomain
			if err := scanOrgDomain(scanner.Scan, &d); err != nil {
				return nil, nil, err
			}
			domains = append(domains, &d)
		}
		counter++
	}
	return domains, paginationClone, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.OrgDomain)
	return p.db.Query(query, domain).Exec()
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	selectQuery := fmt.Sprintf("SELECT id FROM %s WHERE org_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.OrgDomain)
	scanner := p.db.Query(selectQuery, orgID).Iter().Scanner()
	var ids []string
	for scanner.Next() {
		var id string
		if err := scanner.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.OrgDomain)
	for _, id := range ids {
		if err := p.db.Query(deleteQuery, id).Exec(); err != nil {
			return err
		}
	}
	return nil
}
