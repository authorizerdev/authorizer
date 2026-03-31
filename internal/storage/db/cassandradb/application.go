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

// CreateApplication creates a new M2M application
func (p *provider) CreateApplication(ctx context.Context, application *schemas.Application) error {
	if application.ID == "" {
		application.ID = uuid.New().String()
	}
	application.Key = application.ID
	application.CreatedAt = time.Now().Unix()
	application.UpdatedAt = time.Now().Unix()
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.Application)
	err := p.db.Query(insertQuery, application.ID, application.Name, application.Description, application.ClientID, application.ClientSecret, application.Scopes, application.Roles, application.IsActive, application.CreatedBy, application.CreatedAt, application.UpdatedAt).Exec()
	if err != nil {
		return err
	}
	return nil
}

// GetApplicationByID retrieves an application by ID
func (p *provider) GetApplicationByID(ctx context.Context, id string) (*schemas.Application, error) {
	var application schemas.Application
	query := fmt.Sprintf(`SELECT id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at FROM %s WHERE id = ? LIMIT 1`, KeySpace+"."+schemas.Collections.Application)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(&application.ID, &application.Name, &application.Description, &application.ClientID, &application.ClientSecret, &application.Scopes, &application.Roles, &application.IsActive, &application.CreatedBy, &application.CreatedAt, &application.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// GetApplicationByClientID retrieves an application by client ID
func (p *provider) GetApplicationByClientID(ctx context.Context, clientID string) (*schemas.Application, error) {
	var application schemas.Application
	query := fmt.Sprintf(`SELECT id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at FROM %s WHERE client_id = ? LIMIT 1 ALLOW FILTERING`, KeySpace+"."+schemas.Collections.Application)
	err := p.db.Query(query, clientID).Consistency(gocql.One).Scan(&application.ID, &application.Name, &application.Description, &application.ClientID, &application.ClientSecret, &application.Scopes, &application.Roles, &application.IsActive, &application.CreatedBy, &application.CreatedAt, &application.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// ListApplications lists all applications with pagination
func (p *provider) ListApplications(ctx context.Context, pagination *model.Pagination) ([]*schemas.Application, *model.Pagination, error) {
	applications := []*schemas.Application{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.Application)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT id, name, description, client_id, client_secret, scopes, roles, is_active, created_by, created_at, updated_at FROM %s LIMIT %d", KeySpace+"."+schemas.Collections.Application, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var application schemas.Application
			err := scanner.Scan(&application.ID, &application.Name, &application.Description, &application.ClientID, &application.ClientSecret, &application.Scopes, &application.Roles, &application.IsActive, &application.CreatedBy, &application.CreatedAt, &application.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			applications = append(applications, &application)
		}
		counter++
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
	convertMapValues(applicationMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range applicationMap {
		if key == "_id" {
			continue
		}
		if key == "_key" {
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
	updateValues = append(updateValues, application.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Application, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication deletes an application by ID
func (p *provider) DeleteApplication(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Application)
	err := p.db.Query(query, id).Exec()
	if err != nil {
		return err
	}
	return nil
}
