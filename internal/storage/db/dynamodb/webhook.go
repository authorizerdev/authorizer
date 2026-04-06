package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}
	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	if err := p.putItem(ctx, schemas.Collections.Webhook, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	if !strings.Contains(webhook.EventName, "-") {
		webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	}
	if err := p.updateByHashKey(ctx, schemas.Collections.Webhook, "id", webhook.ID, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var webhooks []*schemas.Webhook

	count, err := p.scanCount(ctx, schemas.Collections.Webhook, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.Webhook, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var w schemas.Webhook
			if err := unmarshalItem(it, &w); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				webhooks = append(webhooks, &w)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return webhooks, paginationClone, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	var webhook schemas.Webhook
	err := p.getItemByHash(ctx, schemas.Collections.Webhook, "id", webhookID, &webhook)
	if err != nil {
		return nil, err
	}
	if webhook.ID == "" {
		return nil, errors.New("no document found")
	}
	return &webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	// Match SQL LIKE 'eventName%' (see sql/webhook.go); do not use Contains (substring match).
	f := expression.Name("event_name").BeginsWith(eventName)
	items, err := p.scanFilteredAll(ctx, schemas.Collections.Webhook, strPtr("event_name"), &f)
	if err != nil {
		return nil, err
	}
	var out []*schemas.Webhook
	for _, it := range items {
		var w schemas.Webhook
		if err := unmarshalItem(it, &w); err != nil {
			return nil, err
		}
		out = append(out, &w)
	}
	return out, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	if webhook == nil {
		return nil
	}
	if err := p.deleteItemByHash(ctx, schemas.Collections.Webhook, "id", webhook.ID); err != nil {
		return err
	}
	logs, _, err := p.ListWebhookLogs(ctx, &model.Pagination{}, webhook.ID)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("failed to list webhook logs")
		return nil
	}
	for _, wl := range logs {
		if err := p.deleteItemByHash(ctx, schemas.Collections.WebhookLog, "id", wl.ID); err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("failed to delete webhook log")
		}
	}
	return nil
}
