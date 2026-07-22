package sql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddClient creates a new service account record.
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	if sa.ClientID == "" {
		sa.ClientID = sa.ID
	}
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	res := p.db.Create(sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return sa, nil
}

// UpdateClient updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	res := p.db.Save(sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return sa, nil
}

// DeleteClient removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern. Both deletes run
// in a single transaction so a failure cannot orphan trusted issuers.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("client_id = ?", sa.ID).Delete(&schemas.TrustedIssuer{}).Error; err != nil {
			return err
		}
		return tx.Delete(sa).Error
	})
	if err != nil {
		p.dependencies.Log.Warn().Err(err).Str("client_id", sa.ID).Msg("DeleteClient: cascade transaction failed, rolled back")
	}
	return err
}

// GetClientByID fetches a service account by primary key.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	var sa schemas.Client
	res := p.db.Where("id = ?", id).First(&sa)
	if res.Error != nil {
		return nil, res.Error
	}
	return &sa, nil
}

// GetClientByClientID fetches a client by its unique public client_id.
func (p *provider) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	var sa schemas.Client
	res := p.db.Where("client_id = ?", clientID).First(&sa)
	if res.Error != nil {
		// No matching row is a normal negative result, not a storage failure —
		// callers (e.g. clientauth.ResolveClient) distinguish "no such client"
		// from "couldn't check" by whether err is nil, so a genuinely absent
		// row must come back as (nil, nil), never a wrapped ErrRecordNotFound.
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, res.Error
	}
	return &sa, nil
}

// ListClients returns a paginated list of service accounts.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	var sas []*schemas.Client
	res := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&sas)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	countRes := p.db.Model(&schemas.Client{}).Count(&total)
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
