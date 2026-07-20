package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const samlServiceProviderColumns = "_id, org_id, name, entity_id, acs_url, sp_cert_pem, name_id_format, mapped_attributes, allow_idp_initiated, is_active, created_at, updated_at"

const samlIDPKeyColumns = "_id, org_id, cert_pem, private_key_enc, algorithm, status, created_at, updated_at"

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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(sp)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.SAMLServiceProvider).Insert(sp.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// UpdateSAMLServiceProvider writes back a fully-loaded record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	spMap, err := structToDocument(sp)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(spMap)
	params["_id"] = sp.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.SAMLServiceProvider, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.SAMLServiceProvider).Remove(sp.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetSAMLServiceProviderByID fetches a registered SP by primary key.
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, samlServiceProviderColumns, p.scopeName, schemas.Collections.SAMLServiceProvider)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	sp := &schemas.SAMLServiceProvider{}
	if err := decodeDocument(raw, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// GetSAMLServiceProviderByOrgAndEntityID resolves the single registered SP for an
// (orgID, entityID) pair — the AuthnRequest-Issuer → trusted-ACS binding.
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	params := make(map[string]interface{}, 2)
	params["org_id"] = orgID
	params["entity_id"] = entityID
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id AND entity_id=$entity_id LIMIT 1`, samlServiceProviderColumns, p.scopeName, schemas.Collections.SAMLServiceProvider)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	sp := &schemas.SAMLServiceProvider{}
	if err := decodeDocument(raw, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// ListSAMLServiceProviders returns the registered SPs for an org (paginated).
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	sps := []*schemas.SAMLServiceProvider{}
	paginationClone := pagination
	table := fmt.Sprintf("%s.%s", p.scopeName, schemas.Collections.SAMLServiceProvider)

	params := make(map[string]interface{}, 3)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	params["org_id"] = orgID

	countParams := make(map[string]interface{}, 1)
	countParams["org_id"] = orgID
	total := TotalDocs{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s WHERE org_id=$org_id", table)
	countRes, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: countParams,
	})
	if err != nil {
		return nil, nil, err
	}
	_ = countRes.One(&total)
	paginationClone.Total = total.Total

	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id=$org_id ORDER BY created_at DESC OFFSET $offset LIMIT $limit", samlServiceProviderColumns, table)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, nil, err
		}
		sp := &schemas.SAMLServiceProvider{}
		if err := decodeDocument(raw, sp); err != nil {
			return nil, nil, err
		}
		sps = append(sps, sp)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(key)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.SAMLIDPKey).Insert(key.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// UpdateSAMLIDPKey writes back a fully-loaded record (used to flip status).
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	keyMap, err := structToDocument(key)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(keyMap)
	params["_id"] = key.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.SAMLIDPKey, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteSAMLIDPKey removes a signing key.
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.SAMLIDPKey).Remove(key.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetSAMLIDPKeyByID fetches a signing key by primary key.
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, samlIDPKeyColumns, p.scopeName, schemas.Collections.SAMLIDPKey)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	key := &schemas.SAMLIDPKey{}
	if err := decodeDocument(raw, key); err != nil {
		return nil, err
	}
	return key, nil
}

// ListSAMLIDPKeys returns every signing key for an org (newest first).
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	keys := []*schemas.SAMLIDPKey{}
	params := make(map[string]interface{}, 1)
	params["org_id"] = orgID
	query := fmt.Sprintf("SELECT %s FROM %s.%s WHERE org_id=$org_id ORDER BY created_at DESC", samlIDPKeyColumns, p.scopeName, schemas.Collections.SAMLIDPKey)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, err
		}
		key := &schemas.SAMLIDPKey{}
		if err := decodeDocument(raw, key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}
