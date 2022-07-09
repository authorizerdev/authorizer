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
func (p *provider) AddWebhook(webhook models.Webhook) (models.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	webhookCollection, _ := p.db.Collection(nil, models.Collections.Webhook)
	_, err := webhookCollection.CreateDocument(nil, webhook)
	if err != nil {
		return webhook, err
	}
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(webhook models.Webhook) (models.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	webhookCollection, _ := p.db.Collection(nil, models.Collections.Webhook)
	meta, err := webhookCollection.UpdateDocument(nil, webhook.Key, webhook)
	if err != nil {
		return webhook, err
	}

	webhook.Key = meta.Key
	webhook.ID = meta.ID.String()
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(pagination model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}

	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", models.Collections.Webhook, pagination.Offset, pagination.Limit)

	ctx := driver.WithQueryFullCount(context.Background())
	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()

	for {
		var webhook models.Webhook
		meta, err := cursor.ReadDocument(nil, &webhook)

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
func (p *provider) GetWebhookByID(webhookID string) (models.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf("FOR d in %s FILTER d._id == @webhook_id RETURN d", models.Collections.Webhook)
	bindVars := map[string]interface{}{
		"webhook_id": webhookID,
	}

	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		return webhook, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if webhook.Key == "" {
				return webhook, fmt.Errorf("webhook not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &webhook)
		if err != nil {
			return webhook, err
		}
	}
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(eventName string) (models.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf("FOR d in %s FILTER d.event_name == @event_name RETURN d", models.Collections.Webhook)
	bindVars := map[string]interface{}{
		"event_name": eventName,
	}

	cursor, err := p.db.Query(nil, query, bindVars)
	if err != nil {
		return webhook, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if webhook.Key == "" {
				return webhook, fmt.Errorf("webhook not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &webhook)
		if err != nil {
			return webhook, err
		}
	}
	return webhook, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(webhook models.Webhook) error {
	webhookCollection, _ := p.db.Collection(nil, models.Collections.Webhook)
	_, err := webhookCollection.RemoveDocument(nil, webhook.Key)
	if err != nil {
		return err
	}
	return nil
}
