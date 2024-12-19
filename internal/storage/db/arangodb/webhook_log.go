package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*model.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
		webhookLog.Key = webhookLog.ID
	}
	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()
	webhookLogCollection, _ := p.db.Collection(ctx, schemas.Collections.WebhookLog)
	_, err := webhookLogCollection.CreateDocument(ctx, webhookLog)
	if err != nil {
		return nil, err
	}
	return webhookLog.AsAPIWebhookLog(), nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) (*model.WebhookLogs, error) {
	webhookLogs := []*model.WebhookLog{}
	bindVariables := map[string]interface{}{}
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.WebhookLog, pagination.Offset, pagination.Limit)
	if webhookID != "" {
		query = fmt.Sprintf("FOR d in %s FILTER d.webhook_id == @webhook_id SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.WebhookLog, pagination.Offset, pagination.Limit)
		bindVariables = map[string]interface{}{
			"webhook_id": webhookID,
		}
	}
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, bindVariables)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var webhookLog *schemas.WebhookLog
		meta, err := cursor.ReadDocument(ctx, &webhookLog)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		if meta.Key != "" {
			webhookLogs = append(webhookLogs, webhookLog.AsAPIWebhookLog())
		}
	}
	return &model.WebhookLogs{
		Pagination:  paginationClone,
		WebhookLogs: webhookLogs,
	}, nil
}
