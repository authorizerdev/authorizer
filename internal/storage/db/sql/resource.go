package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

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
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&resource)
	if res.Error != nil {
		return nil, res.Error
	}
	return resource, nil
}

// UpdateResource updates an existing authorization resource.
func (p *provider) UpdateResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	resource.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&resource)
	if result.Error != nil {
		return nil, result.Error
	}
	return resource, nil
}

// DeleteResource deletes an authorization resource by ID.
// Returns an error if any permission references this resource.
func (p *provider) DeleteResource(ctx context.Context, id string) error {
	var count int64
	p.db.Model(&schemas.Permission{}).Where("resource_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("cannot delete resource: %d permission(s) reference it", count)
	}
	result := p.db.Where("id = ?", id).Delete(&schemas.Resource{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetResourceByID returns an authorization resource by its ID.
func (p *provider) GetResourceByID(ctx context.Context, id string) (*schemas.Resource, error) {
	var resource schemas.Resource
	result := p.db.Where("id = ?", id).First(&resource)
	if result.Error != nil {
		return nil, result.Error
	}
	return &resource, nil
}

// GetResourceByName returns an authorization resource by its unique name.
func (p *provider) GetResourceByName(ctx context.Context, name string) (*schemas.Resource, error) {
	var resource schemas.Resource
	result := p.db.Where("name = ?", name).First(&resource)
	if result.Error != nil {
		return nil, result.Error
	}
	return &resource, nil
}

// ListResources returns a paginated list of authorization resources.
func (p *provider) ListResources(ctx context.Context, pagination *model.Pagination) ([]*schemas.Resource, *model.Pagination, error) {
	var resources []*schemas.Resource
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&resources)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.Model(&schemas.Resource{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total
	return resources, paginationClone, nil
}
