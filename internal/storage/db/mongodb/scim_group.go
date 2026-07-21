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

// AddScimGroup creates a new SCIM group record.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.Key = group.ID
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	groupCollection := p.db.Collection(schemas.Collections.ScimGroup, options.Collection())
	_, err := groupCollection.InsertOne(ctx, group)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateScimGroup updates a SCIM group record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	group.UpdatedAt = time.Now().Unix()
	groupCollection := p.db.Collection(schemas.Collections.ScimGroup, options.Collection())
	_, err := groupCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": group.ID}}, bson.M{"$set": group}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return group, nil
}

// DeleteScimGroup removes a SCIM group record.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	groupCollection := p.db.Collection(schemas.Collections.ScimGroup, options.Collection())
	_, err := groupCollection.DeleteOne(ctx, bson.M{"_id": group.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetScimGroupByID fetches a SCIM group by primary key.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	var group *schemas.ScimGroup
	groupCollection := p.db.Collection(schemas.Collections.ScimGroup, options.Collection())
	err := groupCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	var group *schemas.ScimGroup
	groupCollection := p.db.Collection(schemas.Collections.ScimGroup, options.Collection())
	err := groupCollection.FindOne(ctx, bson.M{"org_id": orgID, "display_name": displayName}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId within an org. externalId is stored org-namespaced ("<orgID>:<raw>")
// exactly like User.ExternalID, so this can never resolve another org's group.
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	var group *schemas.ScimGroup
	groupCollection := p.db.Collection(schemas.Collections.ScimGroup, options.Collection())
	err := groupCollection.FindOne(ctx, bson.M{"org_id": orgID, "external_id": orgID + ":" + externalID}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return group, nil
}
