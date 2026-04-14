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

// AddResource creates a new authorization resource.
func (p *provider) AddResource(ctx context.Context, resource *schemas.Resource) (*schemas.Resource, error) {
	if resource.ID == "" {
		resource.ID = uuid.New().String()
	}
	resource.Key = resource.ID
	resource.CreatedAt = time.Now().Unix()
	resource.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Resource).Insert(resource.ID, resource, &insertOpt)
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
	updateFields, params := GetSetFields(resourceMap)
	params["_id"] = resource.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Resource, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// DeleteResource deletes an authorization resource by ID.
// Returns an error if any permission references this resource.
func (p *provider) DeleteResource(ctx context.Context, id string) error {
	// Check for permission references
	params := make(map[string]interface{}, 1)
	params["resource_id"] = id
	query := fmt.Sprintf(`SELECT COUNT(*) as Total FROM %s.%s WHERE resource_id=$resource_id`, p.scopeName, schemas.Collections.Permission)
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
		return fmt.Errorf("cannot delete resource: %d permission(s) reference it", totalDocs.Total)
	}
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err = p.db.Collection(schemas.Collections.Resource).Remove(id, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetResourceByID returns an authorization resource by its ID.
func (p *provider) GetResourceByID(ctx context.Context, id string) (*schemas.Resource, error) {
	var resource *schemas.Resource
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT _id, name, description, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, schemas.Collections.Resource)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&resource)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// GetResourceByName returns an authorization resource by its unique name.
func (p *provider) GetResourceByName(ctx context.Context, name string) (*schemas.Resource, error) {
	var resource *schemas.Resource
	params := make(map[string]interface{}, 1)
	params["name"] = name
	query := fmt.Sprintf(`SELECT _id, name, description, created_at, updated_at FROM %s.%s WHERE name=$name LIMIT 1`, p.scopeName, schemas.Collections.Resource)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&resource)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// ListResources returns a paginated list of authorization resources.
func (p *provider) ListResources(ctx context.Context, pagination *model.Pagination) ([]*schemas.Resource, *model.Pagination, error) {
	resources := []*schemas.Resource{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Resource)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, name, description, created_at, updated_at FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.Resource)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var resource schemas.Resource
		err := queryResult.Row(&resource)
		if err != nil {
			log.Fatal(err)
		}
		resources = append(resources, &resource)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return resources, paginationClone, nil
}
