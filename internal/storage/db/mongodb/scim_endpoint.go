package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimEndpoint creates a new SCIM endpoint record. The unique index on
// org_id rejects a second endpoint for the same organization at the database
// layer.
func (p *provider) AddScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if scimEndpoint.ID == "" {
		scimEndpoint.ID = uuid.New().String()
	}
	scimEndpoint.Key = scimEndpoint.ID
	now := time.Now().Unix()
	scimEndpoint.CreatedAt = now
	scimEndpoint.UpdatedAt = now
	scimEndpointCollection := p.db.Collection(schemas.Collections.ScimEndpoint, options.Collection())
	_, err := scimEndpointCollection.InsertOne(ctx, scimEndpoint)
	if err != nil {
		return nil, err
	}
	return scimEndpoint, nil
}

// UpdateScimEndpoint updates a SCIM endpoint record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if scimEndpoint.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	scimEndpoint.UpdatedAt = time.Now().Unix()
	scimEndpointCollection := p.db.Collection(schemas.Collections.ScimEndpoint, options.Collection())
	_, err := scimEndpointCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": scimEndpoint.ID}}, bson.M{"$set": scimEndpoint}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return scimEndpoint, nil
}

// DeleteScimEndpoint removes a SCIM endpoint record.
func (p *provider) DeleteScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) error {
	scimEndpointCollection := p.db.Collection(schemas.Collections.ScimEndpoint, options.Collection())
	_, err := scimEndpointCollection.DeleteOne(ctx, bson.M{"_id": scimEndpoint.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetScimEndpointByID fetches a SCIM endpoint by primary key.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	var scimEndpoint *schemas.ScimEndpoint
	scimEndpointCollection := p.db.Collection(schemas.Collections.ScimEndpoint, options.Collection())
	err := scimEndpointCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&scimEndpoint)
	if err != nil {
		return nil, err
	}
	return scimEndpoint, nil
}

// GetScimEndpointByOrgID fetches the SCIM endpoint for an organization.
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	var scimEndpoint *schemas.ScimEndpoint
	scimEndpointCollection := p.db.Collection(schemas.Collections.ScimEndpoint, options.Collection())
	err := scimEndpointCollection.FindOne(ctx, bson.M{"org_id": orgID}).Decode(&scimEndpoint)
	if err != nil {
		return nil, err
	}
	return scimEndpoint, nil
}
