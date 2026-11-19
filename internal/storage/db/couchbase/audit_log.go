package couchbase

import (
	"context"
	"fmt"
	"strings"
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
	doc, err := structToDocument(auditLog)
	if err != nil {
		return err
	}
	_, err = p.db.Collection(schemas.Collections.AuditLog).Insert(auditLog.ID, doc, &insertOpt)
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

	// Every filter the ListAuditLogRequest API accepts is applied here. Leaving
	// any of them out silently returns unfiltered rows: the caller cannot tell a
	// filter was dropped, so an auditor narrowing to one resource or one time
	// window would read the result as authoritative.
	clauses := []string{}
	if action, ok := filter["action"]; ok && action != "" {
		clauses = append(clauses, "action=$action")
		params["action"] = action
	}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		clauses = append(clauses, "actor_id=$actorID")
		params["actorID"] = actorID
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		clauses = append(clauses, "resource_type=$resourceType")
		params["resourceType"] = resourceType
	}
	if resourceID, ok := filter["resource_id"]; ok && resourceID != "" {
		clauses = append(clauses, "resource_id=$resourceID")
		params["resourceID"] = resourceID
	}
	if fromTimestamp, ok := filter["from_timestamp"]; ok {
		clauses = append(clauses, "created_at>=$fromTimestamp")
		params["fromTimestamp"] = fromTimestamp
	}
	if toTimestamp, ok := filter["to_timestamp"]; ok {
		clauses = append(clauses, "created_at<=$toTimestamp")
		params["toTimestamp"] = toTimestamp
	}

	whereClause := ""
	if len(clauses) > 0 {
		whereClause = " WHERE " + strings.Join(clauses, " AND ")
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
