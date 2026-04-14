package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
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
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.Scope)
	err := p.db.Query(insertQuery, scope.ID, scope.Name, scope.Description, scope.CreatedAt, scope.UpdatedAt).Exec()
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
	convertMapValues(scopeMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range scopeMap {
		if key == "_id" || key == "_key" || key == "id" || key == "key" {
			continue
		}
		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}
		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	updateValues = append(updateValues, scope.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Scope, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return scope, nil
}

// DeleteScope deletes an authorization scope by ID.
// Returns an error if any permission_scope references this scope.
func (p *provider) DeleteScope(ctx context.Context, id string) error {
	var count int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE scope_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PermissionScope)
	err := p.db.Query(countQuery, id).Consistency(gocql.One).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete scope: %d permission_scope(s) reference it", count)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Scope)
	err = p.db.Query(query, id).Exec()
	if err != nil {
		return err
	}
	return nil
}

// GetScopeByID returns an authorization scope by its ID.
func (p *provider) GetScopeByID(ctx context.Context, id string) (*schemas.Scope, error) {
	var scope schemas.Scope
	query := fmt.Sprintf("SELECT id, name, description, created_at, updated_at FROM %s WHERE id = ? LIMIT 1",
		KeySpace+"."+schemas.Collections.Scope)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(
		&scope.ID, &scope.Name, &scope.Description, &scope.CreatedAt, &scope.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

// GetScopeByName returns an authorization scope by its unique name.
func (p *provider) GetScopeByName(ctx context.Context, name string) (*schemas.Scope, error) {
	var scope schemas.Scope
	query := fmt.Sprintf("SELECT id, name, description, created_at, updated_at FROM %s WHERE name = ? LIMIT 1 ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.Scope)
	err := p.db.Query(query, name).Consistency(gocql.One).Scan(
		&scope.ID, &scope.Name, &scope.Description, &scope.CreatedAt, &scope.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

// ListScopes returns a paginated list of authorization scopes.
func (p *provider) ListScopes(ctx context.Context, pagination *model.Pagination) ([]*schemas.Scope, *model.Pagination, error) {
	scopes := []*schemas.Scope{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.Scope)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	query := fmt.Sprintf("SELECT id, name, description, created_at, updated_at FROM %s LIMIT %d",
		KeySpace+"."+schemas.Collections.Scope, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var scope schemas.Scope
			err := scanner.Scan(&scope.ID, &scope.Name, &scope.Description, &scope.CreatedAt, &scope.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			scopes = append(scopes, &scope)
		}
		counter++
	}
	return scopes, paginationClone, nil
}
