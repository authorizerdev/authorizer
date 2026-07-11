package dynamodb

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgDomain atomically inserts a verified domain row. The partition key "id"
// holds the normalized domain and the write is a conditional PutItem
// (attribute_not_exists), so first-writer-wins is enforced atomically by
// DynamoDB — unlike the scim_endpoint check-then-insert guard, there is NO
// TOCTOU race. On a lost race we classify the existing row by owning org.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	err := p.putItemIfAbsent(ctx, schemas.Collections.OrgDomain, "id", domain)
	if err != nil {
		var conditional *types.ConditionalCheckFailedException
		if errors.As(err, &conditional) {
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

// GetOrgDomainByDomain fetches a verified domain by its normalized value (the
// partition key).
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	var d schemas.OrgDomain
	if err := p.getItemByHash(ctx, schemas.Collections.OrgDomain, "id", domain, &d); err != nil {
		return nil, err
	}
	if d.ID == "" {
		return nil, errors.New("no document found")
	}
	return &d, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	paginationClone := pagination
	items, err := p.queryEq(ctx, schemas.Collections.OrgDomain, "org_id", "org_id", orgID, nil)
	if err != nil {
		return nil, nil, err
	}
	var domains []*schemas.OrgDomain
	for _, it := range items {
		var d schemas.OrgDomain
		if err := unmarshalItem(it, &d); err != nil {
			return nil, nil, err
		}
		domains = append(domains, &d)
	}
	sort.Slice(domains, func(i, j int) bool { return domains[i].CreatedAt > domains[j].CreatedAt })
	paginationClone.Total = int64(len(domains))

	start := int(pagination.Offset)
	if start >= len(domains) {
		return []*schemas.OrgDomain{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(domains) {
		end = len(domains)
	}
	return domains[start:end], paginationClone, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	return p.deleteItemByHash(ctx, schemas.Collections.OrgDomain, "id", domain)
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	items, err := p.queryEq(ctx, schemas.Collections.OrgDomain, "org_id", "org_id", orgID, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var d schemas.OrgDomain
		if err := unmarshalItem(it, &d); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.OrgDomain, "id", d.ID); err != nil {
			return err
		}
	}
	return nil
}
