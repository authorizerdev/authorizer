package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// CreateApplication creates a new M2M application
func (p *provider) CreateApplication(ctx context.Context, application *schemas.Application) error {
	collection := p.db.Table(schemas.Collections.Application)
	if application.ID == "" {
		application.ID = uuid.New().String()
	}
	application.Key = application.ID
	application.CreatedAt = time.Now().Unix()
	application.UpdatedAt = time.Now().Unix()
	err := collection.Put(application).RunWithContext(ctx)
	if err != nil {
		return err
	}
	return nil
}

// GetApplicationByID retrieves an application by ID
func (p *provider) GetApplicationByID(ctx context.Context, id string) (*schemas.Application, error) {
	collection := p.db.Table(schemas.Collections.Application)
	var application schemas.Application
	err := collection.Get("id", id).OneWithContext(ctx, &application)
	if err != nil {
		return nil, err
	}
	if application.ID == "" {
		return nil, errors.New("no document found")
	}
	return &application, nil
}

// GetApplicationByClientID retrieves an application by client ID
func (p *provider) GetApplicationByClientID(ctx context.Context, clientID string) (*schemas.Application, error) {
	collection := p.db.Table(schemas.Collections.Application)
	var applications []schemas.Application
	err := collection.Scan().Filter("client_id = ?", clientID).AllWithContext(ctx, &applications)
	if err != nil {
		return nil, err
	}
	if len(applications) == 0 {
		return nil, errors.New("no document found")
	}
	return &applications[0], nil
}

// ListApplications lists all applications with pagination
func (p *provider) ListApplications(ctx context.Context, pagination *model.Pagination) ([]*schemas.Application, *model.Pagination, error) {
	applications := []*schemas.Application{}
	var application schemas.Application
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	collection := p.db.Table(schemas.Collections.Application)
	paginationClone := pagination
	scanner := collection.Scan()
	count, err := scanner.Count()
	if err != nil {
		return nil, nil, err
	}
	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &application) {
			if paginationClone.Offset == iteration {
				a := application
				applications = append(applications, &a)
			}
		}
		err = iter.Err()
		if err != nil {
			return nil, nil, err
		}
		lastEval = iter.LastEvaluatedKey()
		iteration += paginationClone.Limit
	}
	paginationClone.Total = count
	return applications, paginationClone, nil
}

// UpdateApplication updates an application
func (p *provider) UpdateApplication(ctx context.Context, application *schemas.Application) error {
	application.UpdatedAt = time.Now().Unix()
	collection := p.db.Table(schemas.Collections.Application)
	err := UpdateByHashKey(collection, "id", application.ID, application)
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication deletes an application by ID
func (p *provider) DeleteApplication(ctx context.Context, id string) error {
	collection := p.db.Table(schemas.Collections.Application)
	err := collection.Delete("id", id).RunWithContext(ctx)
	if err != nil {
		return err
	}
	return nil
}
