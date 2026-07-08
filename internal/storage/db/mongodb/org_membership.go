package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgMembership creates a new membership. The compound unique index on
// (org_id, user_id) rejects duplicates at the database layer.
func (p *provider) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.ID == "" {
		membership.ID = uuid.New().String()
	}
	membership.Key = membership.ID
	now := time.Now().Unix()
	membership.CreatedAt = now
	membership.UpdatedAt = now
	membershipCollection := p.db.Collection(schemas.Collections.OrgMembership, options.Collection())
	_, err := membershipCollection.InsertOne(ctx, membership)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// UpdateOrgMembership updates a membership record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	membership.UpdatedAt = time.Now().Unix()
	membershipCollection := p.db.Collection(schemas.Collections.OrgMembership, options.Collection())
	_, err := membershipCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": membership.ID}}, bson.M{"$set": membership}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// DeleteOrgMembership removes a membership record.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	membershipCollection := p.db.Collection(schemas.Collections.OrgMembership, options.Collection())
	_, err := membershipCollection.DeleteOne(ctx, bson.M{"_id": membership.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	var membership *schemas.OrgMembership
	membershipCollection := p.db.Collection(schemas.Collections.OrgMembership, options.Collection())
	err := membershipCollection.FindOne(ctx, bson.M{"org_id": orgID, "user_id": userID}).Decode(&membership)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// ListOrgMembershipsByOrg returns paginated memberships of an organization.
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, bson.M{"org_id": orgID}, pagination)
}

// ListOrgMembershipsByUser returns paginated memberships held by a user.
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, bson.M{"user_id": userID}, pagination)
}

func (p *provider) listOrgMemberships(ctx context.Context, filter bson.M, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	memberships := []*schemas.OrgMembership{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	membershipCollection := p.db.Collection(schemas.Collections.OrgMembership, options.Collection())
	count, err := membershipCollection.CountDocuments(ctx, filter, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := membershipCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var membership *schemas.OrgMembership
		err := cursor.Decode(&membership)
		if err != nil {
			return nil, nil, err
		}
		memberships = append(memberships, membership)
	}
	return memberships, paginationClone, nil
}
