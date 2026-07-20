package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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
	if err := p.putItem(ctx, schemas.Collections.SAMLServiceProvider, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// UpdateSAMLServiceProvider writes back a fully-loaded record. UpdateItem applies
// a partial SET/REMOVE merge, so callers MUST load the record and mutate it
// before calling — a partial struct blanks untouched columns to zero values.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.SAMLServiceProvider, "id", sp.ID, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	if sp == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.SAMLServiceProvider, "id", sp.ID)
}

// GetSAMLServiceProviderByID fetches a registered SP by primary key.
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	var sp schemas.SAMLServiceProvider
	err := p.getItemByHash(ctx, schemas.Collections.SAMLServiceProvider, "id", id, &sp)
	if err != nil {
		return nil, err
	}
	if sp.ID == "" {
		return nil, errors.New("no document found")
	}
	return &sp, nil
}

// GetSAMLServiceProviderByOrgAndEntityID resolves the single registered SP for an
// (orgID, entityID) pair — the AuthnRequest-Issuer → trusted-ACS binding. Query
// the org_id GSI then filter entity_id in Go.
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	f := expression.Name("entity_id").Equal(expression.Value(entityID))
	items, err := p.queryEq(ctx, schemas.Collections.SAMLServiceProvider, "org_id", "org_id", orgID, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var sp schemas.SAMLServiceProvider
	if err := unmarshalItem(items[0], &sp); err != nil {
		return nil, err
	}
	return &sp, nil
}

// ListSAMLServiceProviders returns the registered SPs for an org (paginated).
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	paginationClone := pagination
	items, err := p.queryEq(ctx, schemas.Collections.SAMLServiceProvider, "org_id", "org_id", orgID, nil)
	if err != nil {
		return nil, nil, err
	}
	var sps []*schemas.SAMLServiceProvider
	for _, it := range items {
		var sp schemas.SAMLServiceProvider
		if err := unmarshalItem(it, &sp); err != nil {
			return nil, nil, err
		}
		sps = append(sps, &sp)
	}

	sort.Slice(sps, func(i, j int) bool { return sps[i].CreatedAt > sps[j].CreatedAt })
	paginationClone.Total = int64(len(sps))

	start := int(pagination.Offset)
	if start >= len(sps) {
		return []*schemas.SAMLServiceProvider{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(sps) {
		end = len(sps)
	}
	return sps[start:end], paginationClone, nil
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
	if err := p.putItem(ctx, schemas.Collections.SAMLIDPKey, key); err != nil {
		return nil, err
	}
	return key, nil
}

// UpdateSAMLIDPKey writes back a fully-loaded record (used to flip status).
// Callers MUST load the record and mutate it before calling.
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.SAMLIDPKey, "id", key.ID, key); err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteSAMLIDPKey removes a signing key.
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	if key == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.SAMLIDPKey, "id", key.ID)
}

// GetSAMLIDPKeyByID fetches a signing key by primary key.
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	var key schemas.SAMLIDPKey
	err := p.getItemByHash(ctx, schemas.Collections.SAMLIDPKey, "id", id, &key)
	if err != nil {
		return nil, err
	}
	if key.ID == "" {
		return nil, errors.New("no document found")
	}
	return &key, nil
}

// ListSAMLIDPKeys returns every signing key for an org (newest first).
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	items, err := p.queryEq(ctx, schemas.Collections.SAMLIDPKey, "org_id", "org_id", orgID, nil)
	if err != nil {
		return nil, err
	}
	var keys []*schemas.SAMLIDPKey
	for _, it := range items {
		var key schemas.SAMLIDPKey
		if err := unmarshalItem(it, &key); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].CreatedAt > keys[j].CreatedAt })
	return keys, nil
}
