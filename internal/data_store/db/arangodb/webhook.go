package arangodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*model.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
		webhook.Key = webhook.ID
	}
	webhook.Key = webhook.ID
	// Add timestamp to make event name unique for legacy version
	webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	webhookCollection, _ := p.db.Collection(ctx, schemas.Collections.Webhook)
	_, err := webhookCollection.CreateDocument(ctx, webhook)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*model.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	// Event is changed
	if !strings.Contains(webhook.EventName, "-") {
		webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	}
	webhookCollection, _ := p.db.Collection(ctx, schemas.Collections.Webhook)
	meta, err := webhookCollection.UpdateDocument(ctx, webhook.Key, webhook)
	if err != nil {
		return nil, err
	}
	webhook.Key = meta.Key
	webhook.ID = meta.ID.String()
	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Webhook, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var webhook *schemas.Webhook
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
		Pagination: paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	var webhook *schemas.Webhook
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @webhook_id RETURN d", schemas.Collections.Webhook)
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
			if webhook == nil {
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
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*model.Webhook, error) {
	query := fmt.Sprintf("FOR d in %s FILTER d.event_name LIKE @event_name RETURN d", schemas.Collections.Webhook)
	bindVars := map[string]interface{}{
		"event_name": eventName + "%",
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	webhooks := []*model.Webhook{}
	for {
		var webhook *schemas.Webhook
		if _, err := cursor.ReadDocument(ctx, &webhook); arangoDriver.IsNoMoreDocuments(err) {
			// We're done
			break
		} else if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook.AsAPIWebhook())
	}
	return webhooks, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	webhookCollection, _ := p.db.Collection(ctx, schemas.Collections.Webhook)
	_, err := webhookCollection.RemoveDocument(ctx, webhook.ID)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("FOR d IN %s FILTER d.webhook_id == @webhook_id REMOVE { _key: d._key } IN %s", schemas.Collections.WebhookLog, schemas.Collections.WebhookLog)
	bindVars := map[string]interface{}{
		"webhook_id": webhook.ID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}
