package surrealdb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgDomain claims a verified domain for an organization. The domain is
// the record id, so a second claim fails on it: re-claiming a domain the same
// organization already holds returns the existing row, and a domain held by
// another organization is schemas.ErrOrgDomainConflict.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	err := p.create(ctx, schemas.Collections.OrgDomain, domain.ID, domain)
	if isRecordExists(err) {
		existing, getErr := p.GetOrgDomainByDomain(ctx, domain.ID)
		if getErr != nil {
			return nil, getErr
		}
		if existing.OrgID == domain.OrgID {
			return existing, nil
		}
		return nil, schemas.ErrOrgDomainConflict
	}
	if err != nil {
		return nil, err
	}
	return domain, nil
}

// GetOrgDomainByDomain to get the claim on a domain
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	return getByID[schemas.OrgDomain](ctx, p, schemas.Collections.OrgDomain, domain, "org domain")
}

// ListOrgDomainsByOrg to list an organization's domains, newest first
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	rows, pag, err := page[schemas.OrgDomain](ctx, p, schemas.Collections.OrgDomain, "org_id = $o", map[string]any{"o": orgID}, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// DeleteOrgDomain to release a domain
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	return p.deleteByID(ctx, schemas.Collections.OrgDomain, domain)
}

// DeleteOrgDomainsByOrg to release every domain of an organization
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE org_id = $o RETURN NONE;",
		map[string]any{"tb": schemas.Collections.OrgDomain, "o": orgID})
}
