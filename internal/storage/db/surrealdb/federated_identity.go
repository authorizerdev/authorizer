package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddFederatedIdentity to save a federated identity in database
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.Key = identity.ID
	now := time.Now().Unix()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.FederatedIdentity, identity.ID, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

// GetFederatedIdentity to get the identity an issuer asserted for a subject within an organization
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	return getWhere[schemas.FederatedIdentity](ctx, p, schemas.Collections.FederatedIdentity, "org_id = $o AND issuer = $i AND subject = $s", "federated identity", map[string]any{"o": orgID, "i": issuer, "s": subject})
}
