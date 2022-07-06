package cassandradb

import (
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(webhook models.Webhook) (models.Webhook, error) {
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(webhook models.Webhook) (models.Webhook, error) {
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(pagination model.Pagination) (*model.Webhooks, error) {
	return nil, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(webhookID string) (models.Webhook, error) {
	return models.Webhook{}, nil
}

// GetWebhookByEvent to get webhook by event_name
func (p *provider) GetWebhookByEvent(eventName string) (models.Webhook, error) {
	return models.Webhook{}, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(webhookID string) error {
	return nil
}
