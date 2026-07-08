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

// AddOrganization creates a new organization record.
func (p *provider) AddOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	org.Key = org.ID
	now := time.Now().Unix()
	org.CreatedAt = now
	org.UpdatedAt = now
	orgCollection := p.db.Collection(schemas.Collections.Organization, options.Collection())
	_, err := orgCollection.InsertOne(ctx, org)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// UpdateOrganization updates an organization record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct.
func (p *provider) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	org.UpdatedAt = time.Now().Unix()
	orgCollection := p.db.Collection(schemas.Collections.Organization, options.Collection())
	_, err := orgCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": org.ID}}, bson.M{"$set": org}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return org, nil
}

// DeleteOrganization removes an organization and all its memberships.
// Mirrors the DeleteClient cascade-delete pattern.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	orgCollection := p.db.Collection(schemas.Collections.Organization, options.Collection())
	_, err := orgCollection.DeleteOne(ctx, bson.M{"_id": org.ID}, options.Delete())
	if err != nil {
		return err
	}
	membershipCollection := p.db.Collection(schemas.Collections.OrgMembership, options.Collection())
	_, err = membershipCollection.DeleteMany(ctx, bson.M{"org_id": org.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetOrganizationByID fetches an organization by primary key.
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	var org *schemas.Organization
	orgCollection := p.db.Collection(schemas.Collections.Organization, options.Collection())
	err := orgCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&org)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// GetOrganizationByName fetches an organization by its unique name slug.
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	var org *schemas.Organization
	orgCollection := p.db.Collection(schemas.Collections.Organization, options.Collection())
	err := orgCollection.FindOne(ctx, bson.M{"name": name}).Decode(&org)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// ListOrganizations returns a paginated list of organizations.
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	orgs := []*schemas.Organization{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	orgCollection := p.db.Collection(schemas.Collections.Organization, options.Collection())
	count, err := orgCollection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := orgCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var org *schemas.Organization
		err := cursor.Decode(&org)
		if err != nil {
			return nil, nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, paginationClone, nil
}
