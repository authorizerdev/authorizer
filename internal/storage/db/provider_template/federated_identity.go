package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddFederatedIdentity records a JIT-provisioned upstream identity. The
// (org_id, issuer, subject) triple is unique — adding a duplicate returns an
// error.
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.CreatedAt = time.Now().Unix()
	identity.UpdatedAt = time.Now().Unix()
	return identity, nil
}

// GetFederatedIdentity fetches the identity for a (orgID, issuer, subject) triple.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	return nil, nil
}
