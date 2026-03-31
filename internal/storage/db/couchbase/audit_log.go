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

// AddAuditLog adds an audit log entry
func (p *provider) AddAuditLog(ctx context.Context, auditLog *schemas.AuditLog) error {
	if auditLog.ID == "" {
		auditLog.ID = uuid.New().String()
	}
	auditLog.Key = auditLog.ID
	if auditLog.Timestamp == 0 {
		auditLog.Timestamp = time.Now().Unix()
	}
	auditLog.CreatedAt = time.Now().Unix()
	auditLog.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.AuditLog).Insert(auditLog.ID, auditLog, &insertOpt)
	if err != nil {
		return err
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	auditLogs := []*schemas.AuditLog{}
	paginationClone := pagination
	params := make(map[string]interface{})
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit

	total, err := p.GetTotalDocs(ctx, schemas.Collections.AuditLog)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total

	whereClause := ""
	if action, ok := filter["action"]; ok && action != "" {
		whereClause += " WHERE action=$action"
		params["action"] = action
	}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		if whereClause == "" {
			whereClause += " WHERE actor_id=$actorID"
		} else {
			whereClause += " AND actor_id=$actorID"
		}
		params["actorID"] = actorID
	}

	query := fmt.Sprintf("SELECT _id, timestamp, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, organization_id, created_at, updated_at FROM %s.%s%s ORDER BY timestamp DESC OFFSET $offset LIMIT $limit",
		p.scopeName, schemas.Collections.AuditLog, whereClause)

	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var auditLog schemas.AuditLog
		err := queryResult.Row(&auditLog)
		if err != nil {
			log.Fatal(err)
		}
		auditLogs = append(auditLogs, &auditLog)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return auditLogs, paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	params := make(map[string]interface{})
	params["before"] = before
	query := fmt.Sprintf("DELETE FROM %s.%s WHERE timestamp < $before",
		p.scopeName, schemas.Collections.AuditLog)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	return err
}
