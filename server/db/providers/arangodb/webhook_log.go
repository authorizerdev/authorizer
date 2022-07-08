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

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(webhookLog models.WebhookLog) (models.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}

	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()
	webhookLogCollection, _ := p.db.Collection(nil, models.Collections.WebhookLog)
	_, err := webhookLogCollection.CreateDocument(nil, webhookLog)
	if err != nil {
		return webhookLog, err
	}
	return webhookLog, nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(pagination model.Pagination, webhookID string) (*model.WebhookLogs, error) {
	webhookLogs := []*model.WebhookLog{}
	bindVariables := map[string]interface{}{}

	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", models.Collections.WebhookLog, pagination.Offset, pagination.Limit)

	if webhookID != "" {
		query = fmt.Sprintf("FOR d in %s FILTER d.webhook_id == @webhookID SORT d.created_at DESC LIMIT %d, %d RETURN d", models.Collections.WebhookLog, pagination.Offset, pagination.Limit)
		bindVariables = map[string]interface{}{
			webhookID: webhookID,
		}
	}
	ctx := driver.WithQueryFullCount(context.Background())
	cursor, err := p.db.Query(ctx, query, bindVariables)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()

	for {
		var webhookLog models.WebhookLog
		meta, err := cursor.ReadDocument(nil, &webhookLog)

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
		Pagination:  &paginationClone,
		WebhookLogs: webhookLogs,
	}, nil
}
