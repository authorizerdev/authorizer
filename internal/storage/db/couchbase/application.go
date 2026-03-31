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

// CreateApplication creates a new M2M application
func (p *provider) CreateApplication(ctx context.Context, application *schemas.Application) error {
	if application.ID == "" {
		application.ID = uuid.New().String()
	}
	application.Key = application.ID
	application.CreatedAt = time.Now().Unix()
	application.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Application).Insert(application.ID, application, &insertOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetApplicationByID retrieves an application by ID
func (p *provider) GetApplicationByID(ctx context.Context, id string) (*schemas.Application, error) {
	var application schemas.Application
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT _id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, schemas.Collections.Application)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&application)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// GetApplicationByClientID retrieves an application by client ID
func (p *provider) GetApplicationByClientID(ctx context.Context, clientID string) (*schemas.Application, error) {
	var application schemas.Application
	params := make(map[string]interface{}, 1)
	params["client_id"] = clientID
	query := fmt.Sprintf(`SELECT _id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at FROM %s.%s WHERE client_id=$client_id LIMIT 1`, p.scopeName, schemas.Collections.Application)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&application)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// ListApplications lists all applications with pagination
func (p *provider) ListApplications(ctx context.Context, pagination *model.Pagination) ([]*schemas.Application, *model.Pagination, error) {
	applications := []*schemas.Application{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Application)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at FROM %s.%s OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.Application)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var application schemas.Application
		err := queryResult.Row(&application)
		if err != nil {
			log.Fatal(err)
		}
		applications = append(applications, &application)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return applications, paginationClone, nil
}

// UpdateApplication updates an application
func (p *provider) UpdateApplication(ctx context.Context, application *schemas.Application) error {
	application.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(application)
	if err != nil {
		return err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	applicationMap := map[string]interface{}{}
	err = decoder.Decode(&applicationMap)
	if err != nil {
		return err
	}
	updateFields, params := GetSetFields(applicationMap)
	params["_id"] = application.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Application, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication deletes an application by ID
func (p *provider) DeleteApplication(ctx context.Context, id string) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Application).Remove(id, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
