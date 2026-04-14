package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	if err := p.putItem(ctx, schemas.Collections.Resource, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

// UpdateResource updates an existing authorization resource.
func (p *provider) UpdateResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	resource.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Resource, "id", resource.ID, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

// DeleteResource deletes an authorization resource by ID.
// Returns an error if any permission references this resource.
func (p *provider) DeleteResource(ctx context.Context, id string) error {
	// Check for referencing permissions via resource_id GSI
	f := expression.Name("resource_id").Equal(expression.Value(id))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.Permission, nil, &f)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return fmt.Errorf("cannot delete resource: %d permission(s) reference it", len(items))
	}
	return p.deleteItemByHash(ctx, schemas.Collections.Resource, "id", id)
}

// GetResourceByID returns an authorization resource by its ID.
func (p *provider) GetResourceByID(ctx context.Context, id string) (*schemas.Resource, error) {
	var resource schemas.Resource
	if err := p.getItemByHash(ctx, schemas.Collections.Resource, "id", id, &resource); err != nil {
		return nil, err
	}
	if resource.ID == "" {
		return nil, errors.New("no document found")
	}
	return &resource, nil
}

// GetResourceByName returns an authorization resource by its unique name.
func (p *provider) GetResourceByName(ctx context.Context, name string) (*schemas.Resource, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.Resource, "name", "name", name, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var resource schemas.Resource
	if err := unmarshalItem(items[0], &resource); err != nil {
		return nil, err
	}
	return &resource, nil
}

// ListResources returns a paginated list of authorization resources.
func (p *provider) ListResources(ctx context.Context, pagination *model.Pagination) ([]*schemas.Resource, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var resources []*schemas.Resource

	count, err := p.scanCount(ctx, schemas.Collections.Resource, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.Resource, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var r schemas.Resource
			if err := unmarshalItem(it, &r); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				resources = append(resources, &r)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return resources, paginationClone, nil
}
