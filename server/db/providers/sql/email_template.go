package sql

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
)

// AddEmailTemplate to add EmailTemplate
func (p *provider) AddEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {
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
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {
	emailTemplate.UpdatedAt = time.Now().Unix()

	res := p.db.Save(&emailTemplate)
	if res.Error != nil {
		return nil, res.Error
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination model.Pagination) (*model.EmailTemplates, error) {
	var emailTemplates []models.EmailTemplate

	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&emailTemplates)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	totalRes := p.db.Model(&models.EmailTemplate{}).Count(&total)
	if totalRes.Error != nil {
		return nil, totalRes.Error
	}

	paginationClone := pagination
	paginationClone.Total = total

	responseEmailTemplates := []*model.EmailTemplate{}
	for _, w := range emailTemplates {
		responseEmailTemplates = append(responseEmailTemplates, w.AsAPIEmailTemplate())
	}
	return &model.EmailTemplates{
		Pagination:     &paginationClone,
		EmailTemplates: responseEmailTemplates,
	}, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*model.EmailTemplate, error) {
	var emailTemplate models.EmailTemplate

	result := p.db.Where("id = ?", emailTemplateID).First(&emailTemplate)
	if result.Error != nil {
		return nil, result.Error
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*model.EmailTemplate, error) {
	var emailTemplate models.EmailTemplate

	result := p.db.Where("event_name = ?", eventName).First(&emailTemplate)
	if result.Error != nil {
		return nil, result.Error
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *model.EmailTemplate) error {
	result := p.db.Delete(&models.EmailTemplate{
		ID: emailTemplate.ID,
	})
	if result.Error != nil {
		return result.Error
	}

	return nil
}
