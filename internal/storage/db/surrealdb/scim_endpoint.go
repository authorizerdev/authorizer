package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimEndpoint to save a SCIM endpoint in database
func (p *provider) AddScimEndpoint(ctx context.Context, v *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	v.Key = v.ID
	now := time.Now().Unix()
	v.CreatedAt = now
	v.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.ScimEndpoint, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateScimEndpoint to update a SCIM endpoint in database. Callers MUST load the record first:
// every field is written.
func (p *provider) UpdateScimEndpoint(ctx context.Context, v *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if v.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	v.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.ScimEndpoint, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DeleteScimEndpoint to delete a SCIM endpoint
func (p *provider) DeleteScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) error {
	return p.deleteByID(ctx, schemas.Collections.ScimEndpoint, scimEndpoint.ID)
}

// GetScimEndpointByID to get a SCIM endpoint by ID
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	return getByID[schemas.ScimEndpoint](ctx, p, schemas.Collections.ScimEndpoint, id, "scim endpoint")
}

// GetScimEndpointByOrgID to get an organization's SCIM endpoint
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	return getWhere[schemas.ScimEndpoint](ctx, p, schemas.Collections.ScimEndpoint, "org_id = $v", "scim endpoint", map[string]any{"v": orgID})
}
