package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// --- SAMLServiceProvider (registered downstream SPs; Authorizer as IdP) ---

// AddSAMLServiceProvider registers a new downstream SP.
func (p *provider) AddSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.ID == "" {
		sp.ID = uuid.New().String()
	}
	sp.Key = sp.ID
	now := time.Now().Unix()
	sp.CreatedAt = now
	sp.UpdatedAt = now
	res := p.db.Create(sp)
	if res.Error != nil {
		return nil, res.Error
	}
	return sp, nil
}

// UpdateSAMLServiceProvider writes back a fully-loaded record. Save writes every
// column, so callers MUST load the record and mutate it before calling.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	res := p.db.Save(sp)
	if res.Error != nil {
		return nil, res.Error
	}
	return sp, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	return p.db.Delete(sp).Error
}

// GetSAMLServiceProviderByID fetches a registered SP by primary key.
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	var sp schemas.SAMLServiceProvider
	res := p.db.Where("id = ?", id).First(&sp)
	if res.Error != nil {
		return nil, res.Error
	}
	return &sp, nil
}

// GetSAMLServiceProviderByOrgAndEntityID resolves the single registered SP for an
// (orgID, entityID) pair — the AuthnRequest-Issuer → trusted-ACS binding.
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	var sp schemas.SAMLServiceProvider
	res := p.db.Where("org_id = ? AND entity_id = ?", orgID, entityID).First(&sp)
	if res.Error != nil {
		return nil, res.Error
	}
	return &sp, nil
}

// ListSAMLServiceProviders returns the registered SPs for an org (paginated).
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	var sps []*schemas.SAMLServiceProvider
	res := p.db.Where("org_id = ?", orgID).Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&sps)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	if err := p.db.Model(&schemas.SAMLServiceProvider{}).Where("org_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, nil, err
	}
	return sps, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}

// --- SAMLIDPKey (per-org signing keypairs with rotation) ---

// AddSAMLIDPKey persists a newly-generated signing keypair.
func (p *provider) AddSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.ID == "" {
		key.ID = uuid.New().String()
	}
	key.Key = key.ID
	now := time.Now().Unix()
	key.CreatedAt = now
	key.UpdatedAt = now
	res := p.db.Create(key)
	if res.Error != nil {
		return nil, res.Error
	}
	return key, nil
}

// UpdateSAMLIDPKey writes back a fully-loaded record (used to flip status).
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	res := p.db.Save(key)
	if res.Error != nil {
		return nil, res.Error
	}
	return key, nil
}

// DeleteSAMLIDPKey removes a signing key.
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	return p.db.Delete(key).Error
}

// GetSAMLIDPKeyByID fetches a signing key by primary key.
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	var key schemas.SAMLIDPKey
	res := p.db.Where("id = ?", id).First(&key)
	if res.Error != nil {
		return nil, res.Error
	}
	return &key, nil
}

// ListSAMLIDPKeys returns every signing key for an org (newest first).
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	var keys []*schemas.SAMLIDPKey
	res := p.db.Where("org_id = ?", orgID).Order("created_at DESC").Find(&keys)
	if res.Error != nil {
		return nil, res.Error
	}
	return keys, nil
}
