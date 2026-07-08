package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const scimEndpointColumns = "_id, org_id, token_hash, enabled, created_at, updated_at"

// AddScimEndpoint creates a new SCIM endpoint. org_id is unique — one endpoint
// per org; Couchbase has no cross-attribute unique constraint, so guard with a
// check-then-insert on org_id (closes the sequential case).
func (p *provider) AddScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	endpoint.Key = endpoint.ID
	now := time.Now().Unix()
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now
	if existing, _ := p.GetScimEndpointByOrgID(ctx, endpoint.OrgID); existing != nil {
		return nil, fmt.Errorf("scim endpoint for org_id %s already exists", endpoint.OrgID)
	}
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(endpoint)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.ScimEndpoint).Insert(endpoint.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return endpoint, nil
}

// UpdateScimEndpoint updates a SCIM endpoint record (token rotation, enable).
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	endpoint.UpdatedAt = time.Now().Unix()
	endpointMap, err := structToDocument(endpoint)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(endpointMap)
	params["_id"] = endpoint.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.ScimEndpoint, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return endpoint, nil
}

// DeleteScimEndpoint removes a SCIM endpoint record.
func (p *provider) DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.ScimEndpoint).Remove(endpoint.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetScimEndpointByID fetches a SCIM endpoint by primary key.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, scimEndpointColumns, p.scopeName, schemas.Collections.ScimEndpoint)
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
	endpoint := &schemas.ScimEndpoint{}
	if err := decodeDocument(raw, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// GetScimEndpointByOrgID fetches an org's SCIM endpoint (org_id is unique).
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	params := make(map[string]interface{}, 1)
	params["org_id"] = orgID
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id LIMIT 1`, scimEndpointColumns, p.scopeName, schemas.Collections.ScimEndpoint)
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
	endpoint := &schemas.ScimEndpoint{}
	if err := decodeDocument(raw, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}
