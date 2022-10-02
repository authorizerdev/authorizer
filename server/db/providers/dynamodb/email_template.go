package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"github.com/guregu/dynamo"
)

// AddEmailTemplate to add EmailTemplate
func (p *provider) AddEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {
	collection := p.db.Table(models.Collections.EmailTemplate)
	if emailTemplate.ID == "" {
		emailTemplate.ID = uuid.New().String()
	}

	emailTemplate.Key = emailTemplate.ID
	emailTemplate.CreatedAt = time.Now().Unix()
	emailTemplate.UpdatedAt = time.Now().Unix()
	err := collection.Put(emailTemplate).RunWithContext(ctx)

	if err != nil {
		return emailTemplate.AsAPIEmailTemplate(), err
	}

	return emailTemplate.AsAPIEmailTemplate(), nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {
	collection := p.db.Table(models.Collections.EmailTemplate)
	emailTemplate.UpdatedAt = time.Now().Unix()
	err := UpdateByHashKey(collection, "id", emailTemplate.ID, emailTemplate)
	if err != nil {
		return emailTemplate.AsAPIEmailTemplate(), err
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination model.Pagination) (*model.EmailTemplates, error) {

	var emailTemplate models.EmailTemplate
	var iter dynamo.PagingIter
	var lastEval dynamo.PagingKey
	var iteration int64 = 0

	collection := p.db.Table(models.Collections.EmailTemplate)
	emailTemplates := []*model.EmailTemplate{}
	paginationClone := pagination
	scanner := collection.Scan()
	count, err := scanner.Count()

	if err != nil {
		return nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &emailTemplate) {
			if paginationClone.Offset == iteration {
				emailTemplates = append(emailTemplates, emailTemplate.AsAPIEmailTemplate())
			}
		}
		lastEval = iter.LastEvaluatedKey()
		iteration += paginationClone.Limit
	}

	paginationClone.Total = count

	return &model.EmailTemplates{
		Pagination:     &paginationClone,
		EmailTemplates: emailTemplates,
	}, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*model.EmailTemplate, error) {
	collection := p.db.Table(models.Collections.EmailTemplate)
	var emailTemplate models.EmailTemplate
	err := collection.Get("id", emailTemplateID).OneWithContext(ctx, &emailTemplate)
	if err != nil {
		return nil, err
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*model.EmailTemplate, error) {
	collection := p.db.Table(models.Collections.EmailTemplate)
	var emailTemplates []models.EmailTemplate
	var emailTemplate models.EmailTemplate

	err := collection.Scan().Filter("'event_name' = ?", eventName).Limit(1).AllWithContext(ctx, &emailTemplates)
	if err != nil {
		return nil, err
	}
	if len(emailTemplates) > 0 {
		emailTemplate = emailTemplates[0]
		return emailTemplate.AsAPIEmailTemplate(), nil
	} else {
		return nil, errors.New("no record found")
	}

}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *model.EmailTemplate) error {
	collection := p.db.Table(models.Collections.EmailTemplate)
	err := collection.Delete("id", emailTemplate.ID).RunWithContext(ctx)

	if err != nil {
		return err
	}

	return nil
}
