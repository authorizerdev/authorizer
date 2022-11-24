package surrealdb

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
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {
	emailTemplate.UpdatedAt = time.Now().Unix()
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination model.Pagination) (*model.EmailTemplates, error) {
	return nil, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*model.EmailTemplate, error) {
	return nil, nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*model.EmailTemplate, error) {
	return nil, nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *model.EmailTemplate) error {
	return nil
}
