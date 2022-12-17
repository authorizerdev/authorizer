package couchbase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog models.WebhookLog) (*model.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}

	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()

	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.WebhookLog).Insert(webhookLog.ID, webhookLog, &insertOpt)
	if err != nil {
		return webhookLog.AsAPIWebhookLog(), err
	}

	return webhookLog.AsAPIWebhookLog(), nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination model.Pagination, webhookID string) (*model.WebhookLogs, error) {
	var query string
	var err error

	webhookLogs := []*model.WebhookLog{}
	params := make(map[string]interface{}, 1)
	scope := p.db.Scope("_default")
	paginationClone := pagination

	params["webhookID"] = webhookID
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit

	_, paginationClone.Total = GetTotalDocs(ctx, scope, models.Collections.WebhookLog)

	if webhookID != "" {
		query = fmt.Sprintf(`SELECT _id, http_status, response, request, webhook_id, created_at, updated_at FROM auth._default.%s WHERE webhook_id=$webhookID`, models.Collections.WebhookLog)
	} else {
		query = fmt.Sprintf("SELECT _id, http_status, response, request, webhook_id, created_at, updated_at FROM auth._default.%s OFFSET $offset LIMIT $limit", models.Collections.WebhookLog)
	}

	queryResult, err := scope.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})

	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var webhookLog models.WebhookLog
		err := queryResult.Row(&webhookLog)
		if err != nil {
			log.Fatal(err)
		}
		webhookLogs = append(webhookLogs, webhookLog.AsAPIWebhookLog())
	}

	if err := queryResult.Err(); err != nil {
		return nil, err

	}
	return &model.WebhookLogs{
		Pagination:  &paginationClone,
		WebhookLogs: webhookLogs,
	}, nil
}
