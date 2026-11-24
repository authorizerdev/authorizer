package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddSAMLServiceProvider to save a SAML service provider in database
func (p *provider) AddSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.ID == "" {
		sp.ID = uuid.New().String()
	}
	sp.Key = sp.ID
	now := time.Now().Unix()
	sp.CreatedAt = now
	sp.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.SAMLServiceProvider, sp.ID, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// UpdateSAMLServiceProvider to update a SAML service provider in database
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.SAMLServiceProvider, sp.ID, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// DeleteSAMLServiceProvider to delete a SAML service provider from database
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	return p.deleteByID(ctx, schemas.Collections.SAMLServiceProvider, sp.ID)
}

// GetSAMLServiceProviderByID to get a SAML service provider by its row ID
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	return getByID[schemas.SAMLServiceProvider](ctx, p, schemas.Collections.SAMLServiceProvider, id, "saml service provider")
}

// GetSAMLServiceProviderByOrgAndEntityID to get an organization's SAML service provider by entity ID
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	return getWhere[schemas.SAMLServiceProvider](ctx, p, schemas.Collections.SAMLServiceProvider, "org_id = $o AND entity_id = $e", "saml service provider", map[string]any{"o": orgID, "e": entityID})
}

// ListSAMLServiceProviders lists an organization's SAML service providers, newest first
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	rows, pag, err := page[schemas.SAMLServiceProvider](ctx, p, schemas.Collections.SAMLServiceProvider, "org_id = $o", map[string]any{"o": orgID}, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// AddSAMLIDPKey to save a SAML IdP signing key in database
func (p *provider) AddSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.ID == "" {
		key.ID = uuid.New().String()
	}
	key.Key = key.ID
	now := time.Now().Unix()
	key.CreatedAt = now
	key.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.SAMLIDPKey, key.ID, key); err != nil {
		return nil, err
	}
	return key, nil
}

// UpdateSAMLIDPKey to update a SAML IdP signing key in database
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.SAMLIDPKey, key.ID, key); err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteSAMLIDPKey to delete a SAML IdP signing key from database
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	return p.deleteByID(ctx, schemas.Collections.SAMLIDPKey, key.ID)
}

// GetSAMLIDPKeyByID to get a SAML IdP signing key by its row ID
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	return getByID[schemas.SAMLIDPKey](ctx, p, schemas.Collections.SAMLIDPKey, id, "saml idp key")
}

// ListSAMLIDPKeys lists an organization's SAML IdP signing keys, newest first
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	rows, err := listNewestFirst[schemas.SAMLIDPKey](ctx, p, schemas.Collections.SAMLIDPKey, "org_id = $o", map[string]any{"o": orgID})
	if err != nil {
		return nil, err
	}
	return ptrs(rows), nil
}
