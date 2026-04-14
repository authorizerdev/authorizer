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

// AddScope creates a new authorization scope.
func (p *provider) AddScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	if scope.ID == "" {
		scope.ID = uuid.New().String()
	}
	scope.Key = scope.ID
	scope.CreatedAt = time.Now().Unix()
	scope.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.Scope)
	meta, err := collection.CreateDocument(ctx, scope)
	if err != nil {
		return nil, err
	}
	scope.Key = meta.Key
	scope.ID = meta.ID.String()
	return scope, nil
}

// UpdateScope updates an existing authorization scope.
func (p *provider) UpdateScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	scope.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.Scope)
	meta, err := collection.UpdateDocument(ctx, scope.Key, scope)
	if err != nil {
		return nil, err
	}
	scope.Key = meta.Key
	scope.ID = meta.ID.String()
	return scope, nil
}

// DeleteScope deletes an authorization scope by ID.
// Returns an error if any permission_scope references this scope.
func (p *provider) DeleteScope(ctx context.Context, id string) error {
	// Check for referencing permission_scopes
	countQuery := fmt.Sprintf("FOR d IN %s FILTER d.scope_id == @scope_id COLLECT WITH COUNT INTO length RETURN length", schemas.Collections.PermissionScope)
	cursor, err := p.db.Query(ctx, countQuery, map[string]interface{}{
		"scope_id": id,
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
		return fmt.Errorf("cannot delete scope: %d permission_scope(s) reference it", count)
	}

	// Find the document key for this scope
	scope, err := p.GetScopeByID(ctx, id)
	if err != nil {
		return err
	}
	collection, _ := p.db.Collection(ctx, schemas.Collections.Scope)
	_, err = collection.RemoveDocument(ctx, scope.Key)
	return err
}

// GetScopeByID returns an authorization scope by its ID.
func (p *provider) GetScopeByID(ctx context.Context, id string) (*schemas.Scope, error) {
	var scope *schemas.Scope
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id RETURN d", schemas.Collections.Scope)
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
			if scope == nil {
				return nil, fmt.Errorf("scope not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &scope)
		if err != nil {
			return nil, err
		}
	}
	return scope, nil
}

// GetScopeByName returns an authorization scope by its unique name.
func (p *provider) GetScopeByName(ctx context.Context, name string) (*schemas.Scope, error) {
	var scope *schemas.Scope
	query := fmt.Sprintf("FOR d IN %s FILTER d.name == @name RETURN d", schemas.Collections.Scope)
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
			if scope == nil {
				return nil, fmt.Errorf("scope not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &scope)
		if err != nil {
			return nil, err
		}
	}
	return scope, nil
}

// ListScopes returns a paginated list of authorization scopes.
func (p *provider) ListScopes(ctx context.Context, pagination *model.Pagination) ([]*schemas.Scope, *model.Pagination, error) {
	scopes := []*schemas.Scope{}
	query := fmt.Sprintf("FOR d IN %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Scope, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var scope *schemas.Scope
		meta, err := cursor.ReadDocument(ctx, &scope)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			scopes = append(scopes, scope)
		}
	}
	return scopes, paginationClone, nil
}
