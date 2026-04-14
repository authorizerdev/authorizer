package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Scope).Insert(scope.ID, scope, &insertOpt)
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// UpdateScope updates an existing authorization scope.
func (p *provider) UpdateScope(ctx context.Context, scope *schemas.Scope) (*schemas.Scope, error) {
	scope.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(scope)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	scopeMap := map[string]interface{}{}
	err = decoder.Decode(&scopeMap)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(scopeMap)
	params["_id"] = scope.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Scope, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// DeleteScope deletes an authorization scope by ID.
// Returns an error if any permission_scope references this scope.
func (p *provider) DeleteScope(ctx context.Context, id string) error {
	// Check for permission_scope references
	params := make(map[string]interface{}, 1)
	params["scope_id"] = id
	query := fmt.Sprintf(`SELECT COUNT(*) as Total FROM %s.%s WHERE scope_id=$scope_id`, p.scopeName, schemas.Collections.PermissionScope)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	var totalDocs TotalDocs
	err = queryResult.One(&totalDocs)
	if err != nil {
		return err
	}
	if totalDocs.Total > 0 {
		return fmt.Errorf("cannot delete scope: %d permission_scope(s) reference it", totalDocs.Total)
	}
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err = p.db.Collection(schemas.Collections.Scope).Remove(id, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetScopeByID returns an authorization scope by its ID.
func (p *provider) GetScopeByID(ctx context.Context, id string) (*schemas.Scope, error) {
	var scope *schemas.Scope
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT _id, name, description, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, schemas.Collections.Scope)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// GetScopeByName returns an authorization scope by its unique name.
func (p *provider) GetScopeByName(ctx context.Context, name string) (*schemas.Scope, error) {
	var scope *schemas.Scope
	params := make(map[string]interface{}, 1)
	params["name"] = name
	query := fmt.Sprintf(`SELECT _id, name, description, created_at, updated_at FROM %s.%s WHERE name=$name LIMIT 1`, p.scopeName, schemas.Collections.Scope)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// ListScopes returns a paginated list of authorization scopes.
func (p *provider) ListScopes(ctx context.Context, pagination *model.Pagination) ([]*schemas.Scope, *model.Pagination, error) {
	scopes := []*schemas.Scope{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Scope)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, name, description, created_at, updated_at FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.Scope)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var scope schemas.Scope
		err := queryResult.Row(&scope)
		if err != nil {
			log.Fatal(err)
		}
		scopes = append(scopes, &scope)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return scopes, paginationClone, nil
}
