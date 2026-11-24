package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhookLog to add a webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*schemas.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}
	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()
	if err := p.create(ctx, schemas.Collections.WebhookLog, webhookLog.ID, webhookLog); err != nil {
		return nil, err
	}
	return webhookLog, nil
}

// ListWebhookLogs to list webhook logs, optionally only one webhook's, newest first
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error) {
	where, vars := "", map[string]any{}
	if webhookID != "" {
		where, vars["w"] = "webhook_id = $w", webhookID
	}
	rows, pag, err := page[schemas.WebhookLog](ctx, p, schemas.Collections.WebhookLog, where, vars, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}
