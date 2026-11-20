package surrealdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhook to add a webhook
func (p *provider) AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}
	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	// Add a timestamp to make the event name unique, as every backend does.
	webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	if err := p.create(ctx, schemas.Collections.Webhook, webhook.ID, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// UpdateWebhook to update a webhook. Callers MUST load the record first: every
// field is written.
func (p *provider) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
	if webhook.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebhook: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	webhook.UpdatedAt = time.Now().Unix()
	// The event was changed: make the new name unique too.
	if !strings.Contains(webhook.EventName, "-") {
		webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	}
	if err := p.replace(ctx, schemas.Collections.Webhook, webhook.ID, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// ListWebhook to list webhooks, newest first
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	rows, pag, err := page[schemas.Webhook](ctx, p, schemas.Collections.Webhook, "", nil, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// GetWebhookByID to get a webhook by ID
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	return getByID[schemas.Webhook](ctx, p, schemas.Collections.Webhook, webhookID, "webhook")
}

// GetWebhookByEventName returns the webhooks for an event, newest first.
// Stored names carry a uniqueness suffix ("<event>-<unix time>"), so this is
// a prefix match; string::starts_with matches the prefix literally.
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	rows, err := listNewestFirst[schemas.Webhook](ctx, p, schemas.Collections.Webhook, "string::starts_with(event_name, $e)", map[string]any{"e": eventName})
	if err != nil {
		return nil, err
	}
	return ptrs(rows), nil
}

// DeleteWebhook deletes a webhook and its logs, atomically.
func (p *provider) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	return p.transaction(ctx, `DELETE type::table($ltb) WHERE webhook_id = $id RETURN NONE;
		DELETE type::record($tb, $id) RETURN NONE;`,
		map[string]any{"ltb": schemas.Collections.WebhookLog, "tb": schemas.Collections.Webhook, "id": webhook.ID})
}
