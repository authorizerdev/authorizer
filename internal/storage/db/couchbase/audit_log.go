package couchbase

import (
	"context"
	"fmt"
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
	if auditLog.CreatedAt == 0 {
		auditLog.CreatedAt = time.Now().Unix()
	}
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
	paginationClone := *pagination
	params := make(map[string]interface{})
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit

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

	// Count with filters applied
	countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s.%s%s",
		p.scopeName, schemas.Collections.AuditLog, whereClause)
	countResult, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	var countRow struct {
		Count int64 `json:"count"`
	}
	if countResult.Next() {
		if err := countResult.Row(&countRow); err != nil {
			return nil, nil, err
		}
	}
	paginationClone.Total = countRow.Count

	query := fmt.Sprintf("SELECT _id, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at FROM %s.%s%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit",
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
			return nil, nil, err
		}
		auditLogs = append(auditLogs, &auditLog)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return auditLogs, &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	params := make(map[string]interface{})
	params["before"] = before
	query := fmt.Sprintf("DELETE FROM %s.%s WHERE created_at < $before",
		p.scopeName, schemas.Collections.AuditLog)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	return err
}
