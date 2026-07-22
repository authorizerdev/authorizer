package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimEndpoint creates a new SCIM endpoint. OrgID is unique — one endpoint per org.
func (p *provider) AddScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	endpoint.CreatedAt = time.Now().Unix()
	endpoint.UpdatedAt = time.Now().Unix()
	return endpoint, nil
}

// GetScimEndpointByID fetches an endpoint by primary key.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	return nil, nil
}

// GetScimEndpointByOrgID fetches an org's endpoint.
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	return nil, nil
}

// UpdateScimEndpoint updates an existing endpoint (token rotation, enable).
// Callers MUST load-then-mutate — Save writes every column.
func (p *provider) UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	endpoint.UpdatedAt = time.Now().Unix()
	return endpoint, nil
}

// DeleteScimEndpoint removes an endpoint.
func (p *provider) DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error {
	return nil
}
