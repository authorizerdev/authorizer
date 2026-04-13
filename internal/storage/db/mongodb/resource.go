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

// AddResource creates a new authorization resource.
func (p *provider) AddResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	if resource.ID == "" {
		resource.ID = uuid.New().String()
	}
	resource.Key = resource.ID
	resource.CreatedAt = time.Now().Unix()
	resource.UpdatedAt = time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	_, err := collection.InsertOne(ctx, resource)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// UpdateResource updates an existing authorization resource.
func (p *provider) UpdateResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	resource.UpdatedAt = time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	_, err := collection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": resource.ID}}, bson.M{"$set": resource}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// DeleteResource deletes an authorization resource by ID.
// Returns an error if any permission references this resource.
func (p *provider) DeleteResource(ctx context.Context, id string) error {
	permissionCollection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	count, err := permissionCollection.CountDocuments(ctx, bson.M{"resource_id": id}, options.Count())
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete resource: %d permission(s) reference it", count)
	}
	collection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	_, err = collection.DeleteOne(ctx, bson.M{"_id": id}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetResourceByID returns an authorization resource by its ID.
func (p *provider) GetResourceByID(ctx context.Context, id string) (*schemas.Resource, error) {
	var resource schemas.Resource
	collection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetResourceByName returns an authorization resource by its unique name.
func (p *provider) GetResourceByName(ctx context.Context, name string) (*schemas.Resource, error) {
	var resource schemas.Resource
	collection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	err := collection.FindOne(ctx, bson.M{"name": name}).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// ListResources returns a paginated list of authorization resources.
func (p *provider) ListResources(ctx context.Context, pagination *model.Pagination) ([]*schemas.Resource, *model.Pagination, error) {
	resources := []*schemas.Resource{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	collection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	count, err := collection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var resource *schemas.Resource
		err := cursor.Decode(&resource)
		if err != nil {
			return nil, nil, err
		}
		resources = append(resources, resource)
	}
	return resources, paginationClone, nil
}
