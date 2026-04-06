package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	if err := p.putItem(ctx, schemas.Collections.EmailTemplate, emailTemplate); err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	emailTemplate.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.EmailTemplate, "id", emailTemplate.ID, emailTemplate); err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var emailTemplates []*schemas.EmailTemplate

	count, err := p.scanCount(ctx, schemas.Collections.EmailTemplate, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.EmailTemplate, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var e schemas.EmailTemplate
			if err := unmarshalItem(it, &e); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				emailTemplates = append(emailTemplates, &e)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return emailTemplates, paginationClone, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error) {
	var e schemas.EmailTemplate
	if err := p.getItemByHash(ctx, schemas.Collections.EmailTemplate, "id", emailTemplateID, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error) {
	// Query the event_name GSI — Scan+Limit applies before filters and can return zero matching items.
	items, err := p.queryEq(ctx, schemas.Collections.EmailTemplate, "event_name", "event_name", eventName, nil)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no record found")
	}
	var e schemas.EmailTemplate
	if err := unmarshalItem(items[0], &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error {
	if emailTemplate == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.EmailTemplate, "id", emailTemplate.ID)
}
