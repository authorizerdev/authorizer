package sql

import (
	"context"
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
	sa.CreatedAt = time.Now().Unix()
	sa.UpdatedAt = time.Now().Unix()
	res := p.db.Create(sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	sa.UpdatedAt = time.Now().Unix()
	res := p.db.Save(sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return sa, nil
}

// DeleteServiceAccount removes a service account record.
func (p *provider) DeleteServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) error {
	res := p.db.Delete(sa)
	return res.Error
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
	p.db.Model(&schemas.ServiceAccount{}).Count(&total)
	return sas, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}
