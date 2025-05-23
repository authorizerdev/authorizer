package sql

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddEmailTemplate to add EmailTemplate
func (p *provider) AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	if emailTemplate.ID == "" {
		emailTemplate.ID = uuid.New().String()
	}

	emailTemplate.Key = emailTemplate.ID
	emailTemplate.CreatedAt = time.Now().Unix()
	emailTemplate.UpdatedAt = time.Now().Unix()

	res := p.db.Create(&emailTemplate)
	if res.Error != nil {
		return nil, res.Error
	}
	return emailTemplate, nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	emailTemplate.UpdatedAt = time.Now().Unix()

	res := p.db.Save(&emailTemplate)
	if res.Error != nil {
		return nil, res.Error
	}
	return emailTemplate, nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error) {
	var emailTemplates []*schemas.EmailTemplate
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&emailTemplates)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	var total int64
	totalRes := p.db.Model(&schemas.EmailTemplate{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}

	paginationClone := pagination
	paginationClone.Total = total

	return emailTemplates, paginationClone, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error) {
	var emailTemplate *schemas.EmailTemplate

	result := p.db.Where("id = ?", emailTemplateID).First(&emailTemplate)
	if result.Error != nil {
		return nil, result.Error
	}
	return emailTemplate, nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error) {
	var emailTemplate *schemas.EmailTemplate

	result := p.db.Where("event_name = ?", eventName).First(&emailTemplate)
	if result.Error != nil {
		return nil, result.Error
	}
	return emailTemplate, nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error {
	result := p.db.Delete(&schemas.EmailTemplate{
		ID: emailTemplate.ID,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
