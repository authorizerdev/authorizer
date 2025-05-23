package provider_template

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	// Add timestamp to make event name unique for legacy version
	webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	// Event is changed
	if !strings.Contains(webhook.EventName, "-") {
		webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	}
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	return nil, nil, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	return nil, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	return nil, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	// Also delete webhook logs for given webhook id
	return nil
}
