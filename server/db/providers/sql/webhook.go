package sql

import (
	"time"

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
	res := p.db.Create(&webhook)
	if res.Error != nil {
		return webhook, res.Error
	}
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(webhook models.Webhook) (models.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()

	result := p.db.Save(&webhook)
	if result.Error != nil {
		return webhook, result.Error
	}

	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(pagination model.Pagination) (*model.Webhooks, error) {
	var webhooks []models.Webhook

	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&webhooks)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	totalRes := p.db.Model(&models.Webhook{}).Count(&total)
	if totalRes.Error != nil {
		return nil, totalRes.Error
	}

	paginationClone := pagination
	paginationClone.Total = total

	responseWebhooks := []*model.Webhook{}
	for _, w := range webhooks {
		responseWebhooks = append(responseWebhooks, w.AsAPIWebhook())
	}
	return &model.Webhooks{
		Pagination: &paginationClone,
		Webhooks:   responseWebhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(webhookID string) (models.Webhook, error) {
	var webhook models.Webhook

	result := p.db.Where("id = ?", webhookID).First(webhook)
	if result.Error != nil {
		return webhook, result.Error
	}
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(eventName string) (models.Webhook, error) {
	var webhook models.Webhook

	result := p.db.Where("event_name = ?", eventName).First(webhook)
	if result.Error != nil {
		return webhook, result.Error
	}
	return models.Webhook{}, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(webhook models.Webhook) error {
	result := p.db.Delete(&webhook)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
