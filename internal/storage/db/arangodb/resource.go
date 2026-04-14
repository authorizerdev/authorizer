package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

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
	collection, _ := p.db.Collection(ctx, schemas.Collections.Resource)
	meta, err := collection.CreateDocument(ctx, resource)
	if err != nil {
		return nil, err
	}
	resource.Key = meta.Key
	resource.ID = meta.ID.String()
	return resource, nil
}

// UpdateResource updates an existing authorization resource.
func (p *provider) UpdateResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	resource.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.Resource)
	meta, err := collection.UpdateDocument(ctx, resource.Key, resource)
	if err != nil {
		return nil, err
	}
	resource.Key = meta.Key
	resource.ID = meta.ID.String()
	return resource, nil
}

// DeleteResource deletes an authorization resource by ID.
// Returns an error if any permission references this resource.
func (p *provider) DeleteResource(ctx context.Context, id string) error {
	// Check for referencing permissions
	countQuery := fmt.Sprintf("FOR d IN %s FILTER d.resource_id == @resource_id COLLECT WITH COUNT INTO length RETURN length", schemas.Collections.Permission)
	cursor, err := p.db.Query(ctx, countQuery, map[string]interface{}{
		"resource_id": id,
	})
	if err != nil {
		return err
	}
	defer cursor.Close()
	var count int64
	if cursor.HasMore() {
		if _, err := cursor.ReadDocument(ctx, &count); err != nil {
			return err
		}
	}
	if count > 0 {
		return fmt.Errorf("cannot delete resource: %d permission(s) reference it", count)
	}

	// Find the document key for this resource
	resource, err := p.GetResourceByID(ctx, id)
	if err != nil {
		return err
	}
	collection, _ := p.db.Collection(ctx, schemas.Collections.Resource)
	_, err = collection.RemoveDocument(ctx, resource.Key)
	return err
}

// GetResourceByID returns an authorization resource by its ID.
func (p *provider) GetResourceByID(ctx context.Context, id string) (*schemas.Resource, error) {
	var resource *schemas.Resource
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id RETURN d", schemas.Collections.Resource)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if resource == nil {
				return nil, fmt.Errorf("resource not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &resource)
		if err != nil {
			return nil, err
		}
	}
	return resource, nil
}

// GetResourceByName returns an authorization resource by its unique name.
func (p *provider) GetResourceByName(ctx context.Context, name string) (*schemas.Resource, error) {
	var resource *schemas.Resource
	query := fmt.Sprintf("FOR d IN %s FILTER d.name == @name RETURN d", schemas.Collections.Resource)
	bindVars := map[string]interface{}{
		"name": name,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if resource == nil {
				return nil, fmt.Errorf("resource not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &resource)
		if err != nil {
			return nil, err
		}
	}
	return resource, nil
}

// ListResources returns a paginated list of authorization resources.
func (p *provider) ListResources(ctx context.Context, pagination *model.Pagination) ([]*schemas.Resource, *model.Pagination, error) {
	resources := []*schemas.Resource{}
	query := fmt.Sprintf("FOR d IN %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Resource, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var resource *schemas.Resource
		meta, err := cursor.ReadDocument(ctx, &resource)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			resources = append(resources, resource)
		}
	}
	return resources, paginationClone, nil
}
