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

// AddResource creates a new authorization resource.
func (p *provider) AddResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	if resource.ID == "" {
		resource.ID = uuid.New().String()
	}
	resource.Key = resource.ID
	resource.CreatedAt = time.Now().Unix()
	resource.UpdatedAt = time.Now().Unix()
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.Resource)
	err := p.db.Query(insertQuery, resource.ID, resource.Name, resource.Description, resource.CreatedAt, resource.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// UpdateResource updates an existing authorization resource.
func (p *provider) UpdateResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	resource.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	resourceMap := map[string]interface{}{}
	err = decoder.Decode(&resourceMap)
	if err != nil {
		return nil, err
	}
	convertMapValues(resourceMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range resourceMap {
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
	updateValues = append(updateValues, resource.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Resource, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// DeleteResource deletes an authorization resource by ID.
// Returns an error if any permission references this resource.
func (p *provider) DeleteResource(ctx context.Context, id string) error {
	var count int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE resource_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.Permission)
	err := p.db.Query(countQuery, id).Consistency(gocql.One).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete resource: %d permission(s) reference it", count)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Resource)
	err = p.db.Query(query, id).Exec()
	if err != nil {
		return err
	}
	return nil
}

// GetResourceByID returns an authorization resource by its ID.
func (p *provider) GetResourceByID(ctx context.Context, id string) (*schemas.Resource, error) {
	var resource schemas.Resource
	query := fmt.Sprintf("SELECT id, name, description, created_at, updated_at FROM %s WHERE id = ? LIMIT 1",
		KeySpace+"."+schemas.Collections.Resource)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(
		&resource.ID, &resource.Name, &resource.Description, &resource.CreatedAt, &resource.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetResourceByName returns an authorization resource by its unique name.
func (p *provider) GetResourceByName(ctx context.Context, name string) (*schemas.Resource, error) {
	var resource schemas.Resource
	query := fmt.Sprintf("SELECT id, name, description, created_at, updated_at FROM %s WHERE name = ? LIMIT 1 ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.Resource)
	err := p.db.Query(query, name).Consistency(gocql.One).Scan(
		&resource.ID, &resource.Name, &resource.Description, &resource.CreatedAt, &resource.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// ListResources returns a paginated list of authorization resources.
func (p *provider) ListResources(ctx context.Context, pagination *model.Pagination) ([]*schemas.Resource, *model.Pagination, error) {
	resources := []*schemas.Resource{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.Resource)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	query := fmt.Sprintf("SELECT id, name, description, created_at, updated_at FROM %s LIMIT %d",
		KeySpace+"."+schemas.Collections.Resource, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var resource schemas.Resource
			err := scanner.Scan(&resource.ID, &resource.Name, &resource.Description, &resource.CreatedAt, &resource.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			resources = append(resources, &resource)
		}
		counter++
	}
	return resources, paginationClone, nil
}
