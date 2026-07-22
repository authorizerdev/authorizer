package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddTrustedIssuer creates a new trusted issuer record.
func (p *provider) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.ID == "" {
		issuer.ID = uuid.New().String()
	}
	issuer.CreatedAt = time.Now().Unix()
	issuer.UpdatedAt = time.Now().Unix()
	return issuer, nil
}

// UpdateTrustedIssuer updates mutable fields.
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	issuer.UpdatedAt = time.Now().Unix()
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	return nil
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	return nil, nil
}

// GetTrustedIssuerByIssuerURL fetches by issuer URL (unique index).
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	return nil, nil
}

// GetTrustedIssuerByOrgIDAndKind fetches the single trusted issuer for an
// organization of a given kind.
func (p *provider) GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID, kind string) (*schemas.TrustedIssuer, error) {
	return nil, nil
}

// ListTrustedIssuers returns trusted issuers filtered by serviceAccountID.
// Pass an empty serviceAccountID to list all issuers.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	return nil, nil, nil
}
