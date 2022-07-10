package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
	arangoDriver "github.com/arangodb/go-driver"
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
	webhookCollection, _ := p.db.Collection(ctx, models.Collections.Webhook)
	_, err := webhookCollection.CreateDocument(ctx, webhook)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	webhookCollection, _ := p.db.Collection(ctx, models.Collections.Webhook)
	meta, err := webhookCollection.UpdateDocument(ctx, webhook.Key, webhook)
	if err != nil {
		return nil, err
	}

	webhook.Key = meta.Key
	webhook.ID = meta.ID.String()
	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}

	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", models.Collections.Webhook, pagination.Offset, pagination.Limit)

	sctx := driver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()

	for {
		var webhook models.Webhook
		meta, err := cursor.ReadDocument(ctx, &webhook)

		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}

		if meta.Key != "" {
			webhooks = append(webhooks, webhook.AsAPIWebhook())
		}
	}

	return &model.Webhooks{
		Pagination: &paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf("FOR d in %s FILTER d._id == @webhook_id RETURN d", models.Collections.Webhook)
	bindVars := map[string]interface{}{
		"webhook_id": webhookID,
	}

	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if webhook.Key == "" {
				return nil, fmt.Errorf("webhook not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &webhook)
		if err != nil {
			return nil, err
		}
	}
	return webhook.AsAPIWebhook(), nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) (*model.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf("FOR d in %s FILTER d.event_name == @event_name RETURN d", models.Collections.Webhook)
	bindVars := map[string]interface{}{
		"event_name": eventName,
	}

	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if webhook.Key == "" {
				return nil, fmt.Errorf("webhook not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &webhook)
		if err != nil {
			return nil, err
		}
	}
	return webhook.AsAPIWebhook(), nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	webhookCollection, _ := p.db.Collection(ctx, models.Collections.Webhook)
	_, err := webhookCollection.RemoveDocument(ctx, webhook.ID)
	if err != nil {
		return err
	}
	return nil
}
