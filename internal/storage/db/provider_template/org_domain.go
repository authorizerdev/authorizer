package provider_template

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgDomain atomically inserts a verified domain row, keyed by the
// normalized domain. Caller MUST set ID and Domain to the normalized domain
// before calling. First-writer-wins: same org already holding the domain is
// idempotent success, a different org owning it returns schemas.ErrOrgDomainConflict.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	return domain, nil
}

// GetOrgDomainByDomain fetches the verified row for a normalized domain.
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	return nil, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	return nil, nil, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	return nil
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade on
// org delete — otherwise the domain becomes permanently unclaimable).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	return nil
}
