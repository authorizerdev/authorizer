package sql

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddFederatedIdentity creates a new federated identity. The composite unique
// index on (org_id, issuer, subject) rejects duplicates at the database layer.
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.Key = identity.ID
	now := time.Now().Unix()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	res := p.db.Create(identity)
	if res.Error != nil {
		return nil, res.Error
	}
	return identity, nil
}

// GetFederatedIdentity fetches the federated identity for a (orgID, issuer, subject) tuple.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	var identity schemas.FederatedIdentity
	res := p.db.Where("org_id = ? AND issuer = ? AND subject = ?", orgID, issuer, subject).First(&identity)
	if res.Error != nil {
		return nil, res.Error
	}
	return &identity, nil
}
