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

// AddAuditLog adds an audit log entry
func (p *provider) AddAuditLog(ctx context.Context, auditLog *schemas.AuditLog) error {
	if auditLog.ID == "" {
		auditLog.ID = uuid.New().String()
	}
	auditLog.Key = auditLog.ID
	if auditLog.CreatedAt == 0 {
		auditLog.CreatedAt = time.Now().Unix()
	}
	collection, _ := p.db.Collection(ctx, schemas.Collections.AuditLog)
	_, err := collection.CreateDocument(ctx, auditLog)
	if err != nil {
		return err
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	auditLogs := []*schemas.AuditLog{}
	bindVariables := map[string]interface{}{}

	filterQuery := ""
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		filterQuery += " FILTER d.actor_id == @actor_id"
		bindVariables["actor_id"] = actorID
	}
	if action, ok := filter["action"]; ok && action != "" {
		filterQuery += " FILTER d.action == @action"
		bindVariables["action"] = action
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		filterQuery += " FILTER d.resource_type == @resource_type"
		bindVariables["resource_type"] = resourceType
	}
	if resourceID, ok := filter["resource_id"]; ok && resourceID != "" {
		filterQuery += " FILTER d.resource_id == @resource_id"
		bindVariables["resource_id"] = resourceID
	}
	if fromTimestamp, ok := filter["from_timestamp"]; ok {
		filterQuery += " FILTER d.created_at >= @from_timestamp"
		bindVariables["from_timestamp"] = fromTimestamp
	}
	if toTimestamp, ok := filter["to_timestamp"]; ok {
		filterQuery += " FILTER d.created_at <= @to_timestamp"
		bindVariables["to_timestamp"] = toTimestamp
	}

	bindVariables["offset"] = pagination.Offset
	bindVariables["limit"] = pagination.Limit
	query := fmt.Sprintf("FOR d in %s%s SORT d.created_at DESC LIMIT @offset, @limit RETURN d", schemas.Collections.AuditLog, filterQuery)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, bindVariables)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()

	paginationClone := *pagination
	paginationClone.Total = cursor.Statistics().FullCount()

	for {
		var auditLog *schemas.AuditLog
		meta, err := cursor.ReadDocument(ctx, &auditLog)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			auditLogs = append(auditLogs, auditLog)
		}
	}

	return auditLogs, &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	query := fmt.Sprintf("FOR d in %s FILTER d.created_at < @before REMOVE d IN %s", schemas.Collections.AuditLog, schemas.Collections.AuditLog)
	_, err := p.db.Query(ctx, query, map[string]interface{}{
		"before": before,
	})
	return err
}
