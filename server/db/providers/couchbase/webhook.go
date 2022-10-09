package couchbase

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination model.Pagination) (*model.Webhooks, error) {
	return nil, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	return nil, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) (*model.Webhook, error) {
	return nil, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	// Also delete webhook logs for given webhook id
	return nil
}
