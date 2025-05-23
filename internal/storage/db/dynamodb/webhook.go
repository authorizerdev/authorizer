package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	collection := p.db.Table(schemas.Collections.Webhook)
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}
	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	// Add timestamp to make event name unique for legacy version
	webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	err := collection.Put(webhook).RunWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	// Event is changed
	if !strings.Contains(webhook.EventName, "-") {
		webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	}
	collection := p.db.Table(schemas.Collections.Webhook)
	err := UpdateByHashKey(collection, "id", webhook.ID, webhook)
	if err != nil {
		return nil, err
	}
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	webhooks := []*schemas.Webhook{}
	var webhook *schemas.Webhook
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	collection := p.db.Table(schemas.Collections.Webhook)
	paginationClone := pagination
	scanner := collection.Scan()
	count, err := scanner.Count()
	if err != nil {
		return nil, nil, err
	}
	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &webhook) {
			if paginationClone.Offset == iteration {
				webhooks = append(webhooks, webhook)
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
	return webhooks, paginationClone, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	collection := p.db.Table(schemas.Collections.Webhook)
	var webhook *schemas.Webhook
	err := collection.Get("id", webhookID).OneWithContext(ctx, &webhook)
	if err != nil {
		return nil, err
	}
	if webhook.ID == "" {
		return nil, errors.New("no document found")
	}
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	webhooks := []*schemas.Webhook{}
	collection := p.db.Table(schemas.Collections.Webhook)
	err := collection.Scan().Index("event_name").Filter("contains(event_name, ?)", eventName).AllWithContext(ctx, &webhooks)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	// Also delete webhook logs for given webhook id
	if webhook != nil {
		webhookCollection := p.db.Table(schemas.Collections.Webhook)
		webhookLogCollection := p.db.Table(schemas.Collections.WebhookLog)
		err := webhookCollection.Delete("id", webhook.ID).RunWithContext(ctx)
		if err != nil {
			return err
		}
		pagination := &model.Pagination{}
		webhookLogs, _, err := p.ListWebhookLogs(ctx, pagination, webhook.ID)
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("failed to list webhook logs")
		} else {
			for _, webhookLog := range webhookLogs {
				err = webhookLogCollection.Delete("id", webhookLog.ID).RunWithContext(ctx)
				if err != nil {
					p.dependencies.Log.Debug().Err(err).Msg("failed to delete webhook log")
					// continue
				}
			}
		}
	}
	return nil
}
