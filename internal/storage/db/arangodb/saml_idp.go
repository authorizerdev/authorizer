package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
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
	spCollection, _ := p.db.Collection(ctx, schemas.Collections.SAMLServiceProvider)
	doc, err := structToDocument(sp)
	if err != nil {
		return nil, err
	}
	meta, err := spCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	sp.Key = meta.Key
	sp.ID = meta.ID.String()
	return sp, nil
}

// UpdateSAMLServiceProvider updates a registered SP record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct, per this
// method's "callers must load record first" contract.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	spCollection, _ := p.db.Collection(ctx, schemas.Collections.SAMLServiceProvider)
	doc, err := structToDocument(sp)
	if err != nil {
		return nil, err
	}
	meta, err := spCollection.UpdateDocument(ctx, sp.Key, doc)
	if err != nil {
		return nil, err
	}
	sp.Key = meta.Key
	sp.ID = meta.ID.String()
	return sp, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	spCollection, _ := p.db.Collection(ctx, schemas.Collections.SAMLServiceProvider)
	_, err := spCollection.RemoveDocument(ctx, sp.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetSAMLServiceProviderByID fetches a registered SP by primary key.
// Filters on _key, not _id: every real caller holds the bare id
// AsAPISAMLServiceProvider exposes, never the full "collection/key" handle.
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	var sp *schemas.SAMLServiceProvider
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.SAMLServiceProvider)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if sp == nil {
				return nil, fmt.Errorf("saml service provider not found")
			}
			break
		}
		s := &schemas.SAMLServiceProvider{}
		if _, err := readDocument(ctx, cursor, s); err != nil {
			return nil, err
		}
		sp = s
	}
	return sp, nil
}

// GetSAMLServiceProviderByOrgAndEntityID resolves the single registered SP for an
// (orgID, entityID) pair — the AuthnRequest-Issuer → trusted-ACS binding.
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	var sp *schemas.SAMLServiceProvider
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id AND d.entity_id == @entity_id LIMIT 1 RETURN d", schemas.Collections.SAMLServiceProvider)
	bindVars := map[string]interface{}{
		"org_id":    orgID,
		"entity_id": entityID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if sp == nil {
				return nil, fmt.Errorf("saml service provider not found")
			}
			break
		}
		s := &schemas.SAMLServiceProvider{}
		if _, err := readDocument(ctx, cursor, s); err != nil {
			return nil, err
		}
		sp = s
	}
	return sp, nil
}

// ListSAMLServiceProviders returns the registered SPs for an org (paginated).
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	sps := []*schemas.SAMLServiceProvider{}
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.SAMLServiceProvider, pagination.Offset, pagination.Limit)
	bindVars := map[string]interface{}{
		"org_id": orgID,
	}
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, bindVars)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		sp := &schemas.SAMLServiceProvider{}
		meta, err := readDocument(ctx, cursor, sp)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			sps = append(sps, sp)
		}
	}
	return sps, paginationClone, nil
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
	keyCollection, _ := p.db.Collection(ctx, schemas.Collections.SAMLIDPKey)
	doc, err := structToDocument(key)
	if err != nil {
		return nil, err
	}
	meta, err := keyCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	key.Key = meta.Key
	key.ID = meta.ID.String()
	return key, nil
}

// UpdateSAMLIDPKey updates a signing key record (used to flip rotation status).
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct, per this
// method's "callers must load record first" contract.
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	keyCollection, _ := p.db.Collection(ctx, schemas.Collections.SAMLIDPKey)
	doc, err := structToDocument(key)
	if err != nil {
		return nil, err
	}
	meta, err := keyCollection.UpdateDocument(ctx, key.Key, doc)
	if err != nil {
		return nil, err
	}
	key.Key = meta.Key
	key.ID = meta.ID.String()
	return key, nil
}

// DeleteSAMLIDPKey removes a signing key.
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	keyCollection, _ := p.db.Collection(ctx, schemas.Collections.SAMLIDPKey)
	_, err := keyCollection.RemoveDocument(ctx, key.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetSAMLIDPKeyByID fetches a signing key by primary key.
// Filters on _key, not _id: every real caller holds the bare id, never the
// full "collection/key" handle.
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	var key *schemas.SAMLIDPKey
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.SAMLIDPKey)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if key == nil {
				return nil, fmt.Errorf("saml idp key not found")
			}
			break
		}
		k := &schemas.SAMLIDPKey{}
		if _, err := readDocument(ctx, cursor, k); err != nil {
			return nil, err
		}
		key = k
	}
	return key, nil
}

// ListSAMLIDPKeys returns every signing key for an org (newest first).
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	keys := []*schemas.SAMLIDPKey{}
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id SORT d.created_at DESC RETURN d", schemas.Collections.SAMLIDPKey)
	bindVars := map[string]interface{}{
		"org_id": orgID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		key := &schemas.SAMLIDPKey{}
		meta, err := readDocument(ctx, cursor, key)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		if meta.Key != "" {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
