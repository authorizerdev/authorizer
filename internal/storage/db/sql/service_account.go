package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddServiceAccount creates a new service account record.
func (p *provider) AddServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	res := p.db.Create(sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateServiceAccount: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	res := p.db.Save(sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return sa, nil
}

// DeleteServiceAccount removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) error {
	if err := p.db.Where("service_account_id = ?", sa.ID).Delete(&schemas.TrustedIssuer{}).Error; err != nil {
		return err
	}
	return p.db.Delete(sa).Error
}

// GetServiceAccountByID fetches a service account by primary key.
func (p *provider) GetServiceAccountByID(ctx context.Context, id string) (*schemas.ServiceAccount, error) {
	var sa schemas.ServiceAccount
	res := p.db.Where("id = ?", id).First(&sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return &sa, nil
}

// ListServiceAccounts returns a paginated list of service accounts.
func (p *provider) ListServiceAccounts(ctx context.Context, pagination *model.Pagination) ([]*schemas.ServiceAccount, *model.Pagination, error) {
	var sas []*schemas.ServiceAccount
	res := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&sas)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	countRes := p.db.Model(&schemas.ServiceAccount{}).Count(&total)
	if countRes.Error != nil {
		return nil, nil, countRes.Error
	}
	return sas, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}
