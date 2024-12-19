package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*model.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}

	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()

	insertQuery := fmt.Sprintf("INSERT INTO %s (id, http_status, response, request, webhook_id, created_at, updated_at) VALUES ('%s', %d,'%s', '%s', '%s', %d, %d)", KeySpace+"."+schemas.Collections.WebhookLog, webhookLog.ID, webhookLog.HttpStatus, webhookLog.Response, webhookLog.Request, webhookLog.WebhookID, webhookLog.CreatedAt, webhookLog.UpdatedAt)
	err := p.db.Query(insertQuery).Exec()
	if err != nil {
		return nil, err
	}
	return webhookLog.AsAPIWebhookLog(), nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) (*model.WebhookLogs, error) {
	webhookLogs := []*model.WebhookLog{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.WebhookLog)
	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT id, http_status, response, request, webhook_id, created_at, updated_at FROM %s LIMIT %d", KeySpace+"."+schemas.Collections.WebhookLog, pagination.Limit+pagination.Offset)
	if webhookID != "" {
		totalCountQuery = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE webhook_id='%s' ALLOW FILTERING`, KeySpace+"."+schemas.Collections.WebhookLog, webhookID)
		query = fmt.Sprintf("SELECT id, http_status, response, request, webhook_id, created_at, updated_at FROM %s WHERE webhook_id = '%s' LIMIT %d ALLOW FILTERING", KeySpace+"."+schemas.Collections.WebhookLog, webhookID, pagination.Limit+pagination.Offset)
	}

	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, err
	}

	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var webhookLog schemas.WebhookLog
			err := scanner.Scan(&webhookLog.ID, &webhookLog.HttpStatus, &webhookLog.Response, &webhookLog.Request, &webhookLog.WebhookID, &webhookLog.CreatedAt, &webhookLog.UpdatedAt)
			if err != nil {
				return nil, err
			}
			webhookLogs = append(webhookLogs, webhookLog.AsAPIWebhookLog())
		}
		counter++
	}

	return &model.WebhookLogs{
		Pagination:  paginationClone,
		WebhookLogs: webhookLogs,
	}, nil
}
