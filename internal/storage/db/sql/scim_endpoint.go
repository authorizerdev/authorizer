package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimEndpoint creates a new SCIM endpoint record.
func (p *provider) AddScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	endpoint.Key = endpoint.ID
	now := time.Now().Unix()
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now
	res := p.db.Create(endpoint)
	if res.Error != nil {
		return nil, res.Error
	}
	return endpoint, nil
}

// UpdateScimEndpoint updates a SCIM endpoint record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct.
func (p *provider) UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	endpoint.UpdatedAt = time.Now().Unix()
	res := p.db.Save(endpoint)
	if res.Error != nil {
		return nil, res.Error
	}
	return endpoint, nil
}

// DeleteScimEndpoint removes a SCIM endpoint.
func (p *provider) DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error {
	return p.db.Delete(endpoint).Error
}

// GetScimEndpointByID fetches a SCIM endpoint by primary key.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	var endpoint schemas.ScimEndpoint
	res := p.db.Where("id = ?", id).First(&endpoint)
	if res.Error != nil {
		return nil, res.Error
	}
	return &endpoint, nil
}

// GetScimEndpointByOrgID fetches a SCIM endpoint by its unique org ID.
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	var endpoint schemas.ScimEndpoint
	res := p.db.Where("org_id = ?", orgID).First(&endpoint)
	if res.Error != nil {
		return nil, res.Error
	}
	return &endpoint, nil
}
