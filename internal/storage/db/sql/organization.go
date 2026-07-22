package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
	res := p.db.Create(org)
	if res.Error != nil {
		return nil, res.Error
	}
	return org, nil
}

// UpdateOrganization updates an organization record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct.
func (p *provider) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	org.UpdatedAt = time.Now().Unix()
	res := p.db.Save(org)
	if res.Error != nil {
		return nil, res.Error
	}
	return org, nil
}

// DeleteOrganization removes an organization and all its memberships.
// Mirrors the DeleteClient cascade-delete pattern. The whole cascade runs in a
// single transaction so a mid-cascade failure cannot orphan memberships or
// domains.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("org_id = ?", org.ID).Delete(&schemas.OrgMembership{}).Error; err != nil {
			return err
		}
		// Cascade verified domains — otherwise the domain becomes permanently
		// unclaimable (it is the unique PK of org_domains).
		if err := tx.Where("org_id = ?", org.ID).Delete(&schemas.OrgDomain{}).Error; err != nil {
			return err
		}
		return tx.Delete(org).Error
	})
}

// GetOrganizationByID fetches an organization by primary key.
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	var org schemas.Organization
	res := p.db.Where("id = ?", id).First(&org)
	if res.Error != nil {
		return nil, res.Error
	}
	return &org, nil
}

// GetOrganizationByName fetches an organization by its unique name slug.
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	var org schemas.Organization
	res := p.db.Where("name = ?", name).First(&org)
	if res.Error != nil {
		return nil, res.Error
	}
	return &org, nil
}

// ListOrganizations returns a paginated list of organizations.
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	var orgs []*schemas.Organization
	res := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&orgs)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	countRes := p.db.Model(&schemas.Organization{}).Count(&total)
	if countRes.Error != nil {
		return nil, nil, countRes.Error
	}
	return orgs, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}
