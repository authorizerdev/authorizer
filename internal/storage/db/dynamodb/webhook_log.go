package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*schemas.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}
	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.WebhookLog, webhookLog); err != nil {
		return nil, err
	}
	return webhookLog, nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error) {
	paginationClone := pagination
	// Non-nil empty slice: callers/tests expect a slice value even when there are no rows.
	webhookLogs := []*schemas.WebhookLog{}

	if webhookID != "" {
		items, err := p.queryEq(ctx, schemas.Collections.WebhookLog, "webhook_id", "webhook_id", webhookID, nil)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var wl schemas.WebhookLog
			if err := unmarshalItem(it, &wl); err != nil {
				return nil, nil, err
			}
			webhookLogs = append(webhookLogs, &wl)
		}
		paginationClone.Total = 0
		return webhookLogs, paginationClone, nil
	}

	var lastKey map[string]types.AttributeValue
	var iteration int64
	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.WebhookLog, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var wl schemas.WebhookLog
			if err := unmarshalItem(it, &wl); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				webhookLogs = append(webhookLogs, &wl)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = 0
	return webhookLogs, paginationClone, nil
}
