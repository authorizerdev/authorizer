package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimEndpoint creates a new SCIM endpoint record. OrgID is unique (one
// endpoint per org); DynamoDB has no cross-attribute unique constraint, so
// guard with a check-then-insert on the org_id GSI (closes the sequential case).
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
	if err := p.putItem(ctx, schemas.Collections.ScimEndpoint, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// UpdateScimEndpoint updates a SCIM endpoint record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge, so a partial struct
// blanks untouched columns to their zero values.
func (p *provider) UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	endpoint.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.ScimEndpoint, "id", endpoint.ID, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// DeleteScimEndpoint removes a SCIM endpoint record.
func (p *provider) DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error {
	if endpoint == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.ScimEndpoint, "id", endpoint.ID)
}

// GetScimEndpointByID fetches a SCIM endpoint by primary key.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	var endpoint schemas.ScimEndpoint
	err := p.getItemByHash(ctx, schemas.Collections.ScimEndpoint, "id", id, &endpoint)
	if err != nil {
		return nil, err
	}
	if endpoint.ID == "" {
		return nil, errors.New("no document found")
	}
	return &endpoint, nil
}

// GetScimEndpointByOrgID fetches an org's SCIM endpoint via the org_id GSI.
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.ScimEndpoint, "org_id", "org_id", orgID, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var endpoint schemas.ScimEndpoint
	if err := unmarshalItem(items[0], &endpoint); err != nil {
		return nil, err
	}
	return &endpoint, nil
}
