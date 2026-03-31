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

// CreateApplication creates a new M2M application
func (p *provider) CreateApplication(ctx context.Context, application *schemas.Application) error {
	if application.ID == "" {
		application.ID = uuid.New().String()
	}
	application.Key = application.ID
	application.CreatedAt = time.Now().Unix()
	application.UpdatedAt = time.Now().Unix()
	applicationCollection, _ := p.db.Collection(ctx, schemas.Collections.Application)
	meta, err := applicationCollection.CreateDocument(ctx, application)
	if err != nil {
		return err
	}
	application.Key = meta.Key
	application.ID = meta.ID.String()
	return nil
}

// GetApplicationByID retrieves an application by ID
func (p *provider) GetApplicationByID(ctx context.Context, id string) (*schemas.Application, error) {
	var application schemas.Application
	query := fmt.Sprintf("FOR d in %s FILTER d._id == @id RETURN d", schemas.Collections.Application)
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
			if application.ID == "" {
				return nil, fmt.Errorf("application not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &application)
		if err != nil {
			return nil, err
		}
	}
	return &application, nil
}

// GetApplicationByClientID retrieves an application by client ID
func (p *provider) GetApplicationByClientID(ctx context.Context, clientID string) (*schemas.Application, error) {
	var application schemas.Application
	query := fmt.Sprintf("FOR d in %s FILTER d.client_id == @client_id RETURN d", schemas.Collections.Application)
	bindVars := map[string]interface{}{
		"client_id": clientID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if application.ID == "" {
				return nil, fmt.Errorf("application not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &application)
		if err != nil {
			return nil, err
		}
	}
	return &application, nil
}

// ListApplications lists all applications with pagination
func (p *provider) ListApplications(ctx context.Context, pagination *model.Pagination) ([]*schemas.Application, *model.Pagination, error) {
	applications := []*schemas.Application{}
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Application, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var application schemas.Application
		meta, err := cursor.ReadDocument(ctx, &application)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			applications = append(applications, &application)
		}
	}
	return applications, paginationClone, nil
}

// UpdateApplication updates an application
func (p *provider) UpdateApplication(ctx context.Context, application *schemas.Application) error {
	application.UpdatedAt = time.Now().Unix()
	applicationCollection, _ := p.db.Collection(ctx, schemas.Collections.Application)
	meta, err := applicationCollection.UpdateDocument(ctx, application.Key, application)
	if err != nil {
		return err
	}
	application.Key = meta.Key
	application.ID = meta.ID.String()
	return nil
}

// DeleteApplication deletes an application by ID
func (p *provider) DeleteApplication(ctx context.Context, id string) error {
	application, err := p.GetApplicationByID(ctx, id)
	if err != nil {
		return err
	}
	applicationCollection, _ := p.db.Collection(ctx, schemas.Collections.Application)
	_, err = applicationCollection.RemoveDocument(ctx, application.Key)
	if err != nil {
		return err
	}
	return nil
}
