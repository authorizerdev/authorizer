package sql

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
	res := p.db.Create(&webhook)
	if res.Error != nil {
		return nil, res.Error
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
	result := p.db.Save(&webhook)
	if result.Error != nil {
		return nil, result.Error
	}
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	var webhooks []*schemas.Webhook
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&webhooks)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.Model(&schemas.Webhook{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total

	return webhooks, paginationClone, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	var webhook *schemas.Webhook

	result := p.db.Where("id = ?", webhookID).First(&webhook)
	if result.Error != nil {
		return nil, result.Error
	}
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	var webhooks []*schemas.Webhook
	result := p.db.Where("event_name LIKE ?", eventName+"%").Find(&webhooks)
	if result.Error != nil {
		return nil, result.Error
	}
	return webhooks, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	result := p.db.Delete(&schemas.Webhook{
		ID: webhook.ID,
	})
	if result.Error != nil {
		return result.Error
	}

	result = p.db.Where("webhook_id = ?", webhook.ID).Delete(&schemas.WebhookLog{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
