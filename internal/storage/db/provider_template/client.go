package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddClient creates a new service account record.
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.CreatedAt = time.Now().Unix()
	sa.UpdatedAt = time.Now().Unix()
	return sa, nil
}

// UpdateClient updates name, description, allowed_scopes, or is_active.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	sa.UpdatedAt = time.Now().Unix()
	return sa, nil
}

// DeleteClient removes a client. Callers must delete associated TrustedIssuers
// before or within the same logical operation.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	return nil
}

// GetClientByID fetches a client by its surrogate primary key.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	return nil, nil
}

// GetClientByClientID fetches a client by its public, unique client_id.
func (p *provider) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	return nil, nil
}

// ListClients returns a paginated list of all clients.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	return nil, nil, nil
}
