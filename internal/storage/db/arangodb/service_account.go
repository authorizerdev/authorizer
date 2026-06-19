package arangodb

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddServiceAccount creates a new service account record.
// TODO(phase1-pr3): implement ArangoDB provider.
func (p *provider) AddServiceAccount(_ context.Context, _ *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	return nil, fmt.Errorf("arangodb: AddServiceAccount not implemented")
}

// UpdateServiceAccount updates a service account record.
// TODO(phase1-pr3): implement ArangoDB provider.
func (p *provider) UpdateServiceAccount(_ context.Context, _ *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	return nil, fmt.Errorf("arangodb: UpdateServiceAccount not implemented")
}

// DeleteServiceAccount removes a service account record.
// TODO(phase1-pr3): implement ArangoDB provider.
func (p *provider) DeleteServiceAccount(_ context.Context, _ *schemas.ServiceAccount) error {
	return fmt.Errorf("arangodb: DeleteServiceAccount not implemented")
}

// GetServiceAccountByID fetches a service account by primary key.
// TODO(phase1-pr3): implement ArangoDB provider.
func (p *provider) GetServiceAccountByID(_ context.Context, _ string) (*schemas.ServiceAccount, error) {
	return nil, fmt.Errorf("arangodb: GetServiceAccountByID not implemented")
}

// ListServiceAccounts returns a paginated list of service accounts.
// TODO(phase1-pr3): implement ArangoDB provider.
func (p *provider) ListServiceAccounts(_ context.Context, _ *model.Pagination) ([]*schemas.ServiceAccount, *model.Pagination, error) {
	return nil, nil, fmt.Errorf("arangodb: ListServiceAccounts not implemented")
}
