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

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	collection := p.db.Table(models.Collections.Webhook)

	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()

	err := collection.Put(webhook).RunWithContext(ctx)

	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	collection := p.db.Table(models.Collections.Webhook)

	webhook.UpdatedAt = time.Now().Unix()
	err := UpdateByHashKey(collection, "id", webhook.ID, webhook)

	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}
	var webhook models.Webhook
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0

	collection := p.db.Table(models.Collections.Webhook)
	paginationClone := pagination
	scanner := collection.Scan()
	count, err := scanner.Count()

	if err != nil {
		return nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &webhook) {
			if paginationClone.Offset == iteration {
				webhooks = append(webhooks, webhook.AsAPIWebhook())
			}
		}
		err = iter.Err()
		if err != nil {
			return nil, err
		}
		lastEval = iter.LastEvaluatedKey()
		iteration += paginationClone.Limit
	}

	paginationClone.Total = count

	return &model.Webhooks{
		Pagination: &paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	collection := p.db.Table(models.Collections.Webhook)
	var webhook models.Webhook

	err := collection.Get("id", webhookID).OneWithContext(ctx, &webhook)

	if err != nil {
		return nil, err
	}

	if webhook.ID == "" {
		return webhook.AsAPIWebhook(), errors.New("no documets found")
	}

	return webhook.AsAPIWebhook(), nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) (*model.Webhook, error) {
	var webhook models.Webhook
	collection := p.db.Table(models.Collections.Webhook)

	iter := collection.Scan().Index("event_name").Filter("'event_name' = ?", eventName).Iter()

	for iter.NextWithContext(ctx, &webhook) {
		return webhook.AsAPIWebhook(), nil
	}

	err := iter.Err()

	if err != nil {
		return webhook.AsAPIWebhook(), err
	}
	return webhook.AsAPIWebhook(), nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	// Also delete webhook logs for given webhook id
	if webhook.ID != "" {
		webhookCollection := p.db.Table(models.Collections.Webhook)
		pagination := model.Pagination{}
		webhookLogCollection := p.db.Table(models.Collections.WebhookLog)
		err := webhookCollection.Delete("id", webhook.ID).RunWithContext(ctx)
		if err != nil {
			return err
		}
		webhookLogs, errIs := p.ListWebhookLogs(ctx, pagination, webhook.ID)

		for _, webhookLog := range webhookLogs.WebhookLogs {
			err = webhookLogCollection.Delete("id", webhookLog.ID).RunWithContext(ctx)
			if err != nil {
				return err
			}
		}
		if errIs != nil {
			return errIs
		}
	}
	return nil
}
