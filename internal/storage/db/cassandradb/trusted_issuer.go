package cassandradb

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddTrustedIssuer creates a new trusted issuer record.
// TODO(phase1-pr3): implement CassandraDB provider.
// DDL required before implementation:
//
//	CREATE TABLE IF NOT EXISTS authorizer_trusted_issuers (
//	  id text PRIMARY KEY,
//	  service_account_id text, name text, issuer_url text,
//	  key_source_type text, jwks_url text, expected_aud text,
//	  subject_claim text, issuer_type text, auth_method text,
//	  is_active boolean, enable_token_review boolean,
//	  kubernetes_api_server_url text,
//	  spiffe_refresh_hint_seconds bigint,
//	  trusted_proxy_header text, trusted_proxy_cidrs text,
//	  created_at bigint, updated_at bigint
//	);
func (p *provider) AddTrustedIssuer(_ context.Context, _ *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("cassandradb: AddTrustedIssuer not implemented")
}

// UpdateTrustedIssuer updates a trusted issuer record.
// TODO(phase1-pr3): implement CassandraDB provider.
func (p *provider) UpdateTrustedIssuer(_ context.Context, _ *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("cassandradb: UpdateTrustedIssuer not implemented")
}

// DeleteTrustedIssuer removes a trusted issuer record.
// TODO(phase1-pr3): implement CassandraDB provider.
func (p *provider) DeleteTrustedIssuer(_ context.Context, _ *schemas.TrustedIssuer) error {
	return fmt.Errorf("cassandradb: DeleteTrustedIssuer not implemented")
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
// TODO(phase1-pr3): implement CassandraDB provider.
func (p *provider) GetTrustedIssuerByID(_ context.Context, _ string) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("cassandradb: GetTrustedIssuerByID not implemented")
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// TODO(phase1-pr3): implement CassandraDB provider.
func (p *provider) GetTrustedIssuerByIssuerURL(_ context.Context, _ string) (*schemas.TrustedIssuer, error) {
	return nil, fmt.Errorf("cassandradb: GetTrustedIssuerByIssuerURL not implemented")
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
// TODO(phase1-pr3): implement CassandraDB provider.
func (p *provider) ListTrustedIssuers(_ context.Context, _ string, _ *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	return nil, nil, fmt.Errorf("cassandradb: ListTrustedIssuers not implemented")
}
