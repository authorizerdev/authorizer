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

// AddScope creates a new authorization scope.
func (p *provider) AddScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	if scope.ID == "" {
		scope.ID = uuid.New().String()
	}
	scope.Key = scope.ID
	scope.CreatedAt = time.Now().Unix()
	scope.UpdatedAt = time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.Scope, options.Collection())
	_, err := collection.InsertOne(ctx, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// UpdateScope updates an existing authorization scope.
func (p *provider) UpdateScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	scope.UpdatedAt = time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.Scope, options.Collection())
	_, err := collection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": scope.ID}}, bson.M{"$set": scope}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// DeleteScope deletes an authorization scope by ID.
// Returns an error if any permission_scope references this scope.
func (p *provider) DeleteScope(ctx context.Context, id string) error {
	permissionScopeCollection := p.db.Collection(schemas.Collections.PermissionScope, options.Collection())
	count, err := permissionScopeCollection.CountDocuments(ctx, bson.M{"scope_id": id}, options.Count())
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete scope: %d permission_scope(s) reference it", count)
	}
	collection := p.db.Collection(schemas.Collections.Scope, options.Collection())
	_, err = collection.DeleteOne(ctx, bson.M{"_id": id}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetScopeByID returns an authorization scope by its ID.
func (p *provider) GetScopeByID(ctx context.Context, id string) (*schemas.Scope, error) {
	var scope schemas.Scope
	collection := p.db.Collection(schemas.Collections.Scope, options.Collection())
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&scope)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

// GetScopeByName returns an authorization scope by its unique name.
func (p *provider) GetScopeByName(ctx context.Context, name string) (*schemas.Scope, error) {
	var scope schemas.Scope
	collection := p.db.Collection(schemas.Collections.Scope, options.Collection())
	err := collection.FindOne(ctx, bson.M{"name": name}).Decode(&scope)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

// ListScopes returns a paginated list of authorization scopes.
func (p *provider) ListScopes(ctx context.Context, pagination *model.Pagination) ([]*schemas.Scope, *model.Pagination, error) {
	scopes := []*schemas.Scope{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	collection := p.db.Collection(schemas.Collections.Scope, options.Collection())
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
		var scope *schemas.Scope
		err := cursor.Decode(&scope)
		if err != nil {
			return nil, nil, err
		}
		scopes = append(scopes, scope)
	}
	return scopes, paginationClone, nil
}
