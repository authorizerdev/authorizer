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

// AddScope creates a new authorization scope.
func (p *provider) AddScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	if scope.ID == "" {
		scope.ID = uuid.New().String()
	}
	scope.Key = scope.ID
	scope.CreatedAt = time.Now().Unix()
	scope.UpdatedAt = time.Now().Unix()
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&scope)
	if res.Error != nil {
		return nil, res.Error
	}
	return scope, nil
}

// UpdateScope updates an existing authorization scope.
func (p *provider) UpdateScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	scope.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&scope)
	if result.Error != nil {
		return nil, result.Error
	}
	return scope, nil
}

// DeleteScope deletes an authorization scope by ID.
// Returns an error if any permission_scope references this scope.
func (p *provider) DeleteScope(ctx context.Context, id string) error {
	var count int64
	p.db.Model(&schemas.PermissionScope{}).Where("scope_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("cannot delete scope: %d permission_scope(s) reference it", count)
	}
	result := p.db.Where("id = ?", id).Delete(&schemas.Scope{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetScopeByID returns an authorization scope by its ID.
func (p *provider) GetScopeByID(ctx context.Context, id string) (*schemas.Scope, error) {
	var scope schemas.Scope
	result := p.db.Where("id = ?", id).First(&scope)
	if result.Error != nil {
		return nil, result.Error
	}
	return &scope, nil
}

// GetScopeByName returns an authorization scope by its unique name.
func (p *provider) GetScopeByName(ctx context.Context, name string) (*schemas.Scope, error) {
	var scope schemas.Scope
	result := p.db.Where("name = ?", name).First(&scope)
	if result.Error != nil {
		return nil, result.Error
	}
	return &scope, nil
}

// ListScopes returns a paginated list of authorization scopes.
func (p *provider) ListScopes(ctx context.Context, pagination *model.Pagination) ([]*schemas.Scope, *model.Pagination, error) {
	var scopes []*schemas.Scope
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&scopes)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.Model(&schemas.Scope{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total
	return scopes, paginationClone, nil
}
