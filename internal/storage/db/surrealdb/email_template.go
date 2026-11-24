package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddEmailTemplate to add an email template
func (p *provider) AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	if emailTemplate.ID == "" {
		emailTemplate.ID = uuid.New().String()
	}
	emailTemplate.Key = emailTemplate.ID
	emailTemplate.CreatedAt = time.Now().Unix()
	emailTemplate.UpdatedAt = time.Now().Unix()
	if err := p.create(ctx, schemas.Collections.EmailTemplate, emailTemplate.ID, emailTemplate); err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// UpdateEmailTemplate to update an email template. Callers MUST load the
// record first: every field is written.
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	if emailTemplate.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateEmailTemplate: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	emailTemplate.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.EmailTemplate, emailTemplate.ID, emailTemplate); err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// ListEmailTemplate to list email templates, newest first
func (p *provider) ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error) {
	rows, pag, err := page[schemas.EmailTemplate](ctx, p, schemas.Collections.EmailTemplate, "", nil, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// GetEmailTemplateByID to get an email template by ID
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error) {
	return getByID[schemas.EmailTemplate](ctx, p, schemas.Collections.EmailTemplate, emailTemplateID, "email template")
}

// GetEmailTemplateByEventName to get the email template for an event
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error) {
	return getWhere[schemas.EmailTemplate](ctx, p, schemas.Collections.EmailTemplate, "event_name = $v", "email template", map[string]any{"v": eventName})
}

// DeleteEmailTemplate to delete an email template
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error {
	return p.deleteByID(ctx, schemas.Collections.EmailTemplate, emailTemplate.ID)
}
