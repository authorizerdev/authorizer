package sql

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgDomain atomically inserts a verified domain row. The domain is the
// primary key, so a duplicate insert fails at the DB (unique PK) — first-writer
// wins with no check-then-insert race. On conflict we classify by owning org:
// same org → idempotent success, different org → ErrOrgDomainConflict.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	res := p.db.Create(domain)
	if res.Error != nil {
		existing, getErr := p.GetOrgDomainByDomain(ctx, domain.ID)
		if getErr == nil && existing != nil {
			if existing.OrgID == domain.OrgID {
				return existing, nil
			}
			return nil, schemas.ErrOrgDomainConflict
		}
		return nil, res.Error
	}
	return domain, nil
}

// GetOrgDomainByDomain fetches a verified domain by its normalized value (PK).
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	var d schemas.OrgDomain
	res := p.db.Where("id = ?", domain).First(&d)
	if res.Error != nil {
		return nil, res.Error
	}
	return &d, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	var domains []*schemas.OrgDomain
	res := p.db.Where("org_id = ?", orgID).Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&domains)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	countRes := p.db.Model(&schemas.OrgDomain{}).Where("org_id = ?", orgID).Count(&total)
	if countRes.Error != nil {
		return nil, nil, countRes.Error
	}
	return domains, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	return p.db.Where("id = ?", domain).Delete(&schemas.OrgDomain{}).Error
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	return p.db.Where("org_id = ?", orgID).Delete(&schemas.OrgDomain{}).Error
}
