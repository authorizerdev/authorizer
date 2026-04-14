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

// AddScope creates a new authorization scope.
func (p *provider) AddScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	if scope.ID == "" {
		scope.ID = uuid.New().String()
	}
	scope.Key = scope.ID
	scope.CreatedAt = time.Now().Unix()
	scope.UpdatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.Scope, scope); err != nil {
		return nil, err
	}
	return scope, nil
}

// UpdateScope updates an existing authorization scope.
func (p *provider) UpdateScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	scope.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Scope, "id", scope.ID, scope); err != nil {
		return nil, err
	}
	return scope, nil
}

// DeleteScope deletes an authorization scope by ID.
// Returns an error if any permission_scope references this scope.
func (p *provider) DeleteScope(ctx context.Context, id string) error {
	f := expression.Name("scope_id").Equal(expression.Value(id))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionScope, nil, &f)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return fmt.Errorf("cannot delete scope: %d permission_scope(s) reference it", len(items))
	}
	return p.deleteItemByHash(ctx, schemas.Collections.Scope, "id", id)
}

// GetScopeByID returns an authorization scope by its ID.
func (p *provider) GetScopeByID(ctx context.Context, id string) (*schemas.Scope, error) {
	var scope schemas.Scope
	if err := p.getItemByHash(ctx, schemas.Collections.Scope, "id", id, &scope); err != nil {
		return nil, err
	}
	if scope.ID == "" {
		return nil, errors.New("no document found")
	}
	return &scope, nil
}

// GetScopeByName returns an authorization scope by its unique name.
func (p *provider) GetScopeByName(ctx context.Context, name string) (*schemas.Scope, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.Scope, "name", "name", name, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var scope schemas.Scope
	if err := unmarshalItem(items[0], &scope); err != nil {
		return nil, err
	}
	return &scope, nil
}

// ListScopes returns a paginated list of authorization scopes.
func (p *provider) ListScopes(ctx context.Context, pagination *model.Pagination) ([]*schemas.Scope, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var scopes []*schemas.Scope

	count, err := p.scanCount(ctx, schemas.Collections.Scope, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.Scope, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var s schemas.Scope
			if err := unmarshalItem(it, &s); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				scopes = append(scopes, &s)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return scopes, paginationClone, nil
}
