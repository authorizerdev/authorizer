package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddFederatedIdentity creates a new federated identity. The compound unique
// index on (org_id, issuer, subject) rejects duplicates at the database layer.
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.Key = identity.ID
	now := time.Now().Unix()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	identityCollection := p.db.Collection(schemas.Collections.FederatedIdentity, options.Collection())
	_, err := identityCollection.InsertOne(ctx, identity)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

// GetFederatedIdentity fetches the federated identity for a (orgID, issuer, subject) triple.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	var identity *schemas.FederatedIdentity
	identityCollection := p.db.Collection(schemas.Collections.FederatedIdentity, options.Collection())
	err := identityCollection.FindOne(ctx, bson.M{"org_id": orgID, "issuer": issuer, "subject": subject}).Decode(&identity)
	if err != nil {
		return nil, err
	}
	return identity, nil
}
