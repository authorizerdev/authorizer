package dynamodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"github.com/guregu/dynamo"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *models.WebhookLog) (*model.WebhookLog, error) {
	collection := p.db.Table(models.Collections.WebhookLog)

	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}

	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()
	err := collection.Put(webhookLog).RunWithContext(ctx)

	if err != nil {
		return nil, err
	}
	return webhookLog.AsAPIWebhookLog(), nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) (*model.WebhookLogs, error) {
	webhookLogs := []*model.WebhookLog{}
	var webhookLog *models.WebhookLog
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	var err error
	var count int64

	collection := p.db.Table(models.Collections.WebhookLog)
	paginationClone := pagination
	scanner := collection.Scan()

	if webhookID != "" {
		iter = scanner.Index("webhook_id").Filter("'webhook_id' = ?", webhookID).Iter()
		for iter.NextWithContext(ctx, &webhookLog) {
			webhookLogs = append(webhookLogs, webhookLog.AsAPIWebhookLog())
		}
		err = iter.Err()
		if err != nil {
			return nil, err
		}
	} else {
		for (paginationClone.Offset + paginationClone.Limit) > iteration {
			iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
			for iter.NextWithContext(ctx, &webhookLog) {
				if paginationClone.Offset == iteration {
					webhookLogs = append(webhookLogs, webhookLog.AsAPIWebhookLog())
				}
			}
			err = iter.Err()
			if err != nil {
				return nil, err
			}
			lastEval = iter.LastEvaluatedKey()
			iteration += paginationClone.Limit
		}
	}

	paginationClone.Total = count
	// paginationClone.Cursor = iter.LastEvaluatedKey()
	return &model.WebhookLogs{
		Pagination:  paginationClone,
		WebhookLogs: webhookLogs,
	}, nil
}
