package sql

import (
	"context"
	"time"

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
	result := p.db.Create(&application)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetApplicationByID retrieves an application by ID
func (p *provider) GetApplicationByID(ctx context.Context, id string) (*schemas.Application, error) {
	var application schemas.Application
	result := p.db.Where("id = ?", id).First(&application)
	if result.Error != nil {
		return nil, result.Error
	}
	return &application, nil
}

// GetApplicationByClientID retrieves an application by client ID
func (p *provider) GetApplicationByClientID(ctx context.Context, clientID string) (*schemas.Application, error) {
	var application schemas.Application
	result := p.db.Where("client_id = ?", clientID).First(&application)
	if result.Error != nil {
		return nil, result.Error
	}
	return &application, nil
}

// ListApplications lists all applications with pagination
func (p *provider) ListApplications(ctx context.Context, pagination *model.Pagination) ([]*schemas.Application, *model.Pagination, error) {
	var applications []*schemas.Application
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&applications)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.Model(&schemas.Application{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total
	return applications, paginationClone, nil
}

// UpdateApplication updates an application
func (p *provider) UpdateApplication(ctx context.Context, application *schemas.Application) error {
	application.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&application)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteApplication deletes an application by ID
func (p *provider) DeleteApplication(ctx context.Context, id string) error {
	result := p.db.Delete(&schemas.Application{
		ID: id,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
