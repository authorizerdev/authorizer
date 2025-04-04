package couchbase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/couchbase/gocb/v2"
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.WebhookLog).Insert(webhookLog.ID, webhookLog, &insertOpt)
	if err != nil {
		return nil, err
	}
	return webhookLog, nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error) {
	var query string
	var err error
	webhookLogs := []*schemas.WebhookLog{}
	params := make(map[string]interface{}, 1)
	paginationClone := pagination
	params["webhookID"] = webhookID
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.WebhookLog)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	if webhookID != "" {
		query = fmt.Sprintf(`SELECT _id, http_status, response, request, webhook_id, created_at, updated_at FROM %s.%s WHERE webhook_id=$webhookID`, p.scopeName, schemas.Collections.WebhookLog)
	} else {
		query = fmt.Sprintf("SELECT _id, http_status, response, request, webhook_id, created_at, updated_at FROM %s.%s OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.WebhookLog)
	}
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var webhookLog schemas.WebhookLog
		err := queryResult.Row(&webhookLog)
		if err != nil {
			log.Fatal(err)
		}
		webhookLogs = append(webhookLogs, &webhookLog)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err

	}
	return webhookLogs, paginationClone, nil
}
