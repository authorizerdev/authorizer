package dynamodb

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddTrustedIssuer creates a new trusted issuer record.
// TODO(phase1-pr3): implement DynamoDB provider.
func (p *provider) AddTrustedIssuer(_ context.Context, _ *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("dynamodb: AddTrustedIssuer not implemented")
}

// UpdateTrustedIssuer updates a trusted issuer record.
// TODO(phase1-pr3): implement DynamoDB provider.
func (p *provider) UpdateTrustedIssuer(_ context.Context, _ *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("dynamodb: UpdateTrustedIssuer not implemented")
}

// DeleteTrustedIssuer removes a trusted issuer record.
// TODO(phase1-pr3): implement DynamoDB provider.
func (p *provider) DeleteTrustedIssuer(_ context.Context, _ *schemas.TrustedIssuer) error {
	return fmt.Errorf("dynamodb: DeleteTrustedIssuer not implemented")
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
// TODO(phase1-pr3): implement DynamoDB provider.
func (p *provider) GetTrustedIssuerByID(_ context.Context, _ string) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("dynamodb: GetTrustedIssuerByID not implemented")
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// TODO(phase1-pr3): implement DynamoDB provider.
func (p *provider) GetTrustedIssuerByIssuerURL(_ context.Context, _ string) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("dynamodb: GetTrustedIssuerByIssuerURL not implemented")
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
// TODO(phase1-pr3): implement DynamoDB provider.
func (p *provider) ListTrustedIssuers(_ context.Context, _ string, _ *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	return nil, nil, fmt.Errorf("dynamodb: ListTrustedIssuers not implemented")
}
