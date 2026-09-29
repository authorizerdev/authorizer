package cassandradb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
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

	insertQuery := fmt.Sprintf("INSERT INTO %s (id, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.AuditLog)
	err := p.db.Query(insertQuery,
		auditLog.ID, auditLog.ActorID, auditLog.ActorType, auditLog.ActorEmail,
		auditLog.Action, auditLog.ResourceType, auditLog.ResourceID, auditLog.IPAddress,
		auditLog.UserAgent, auditLog.Metadata,
		auditLog.CreatedAt).Exec()
	if err != nil {
		return err
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	auditLogs := []*schemas.AuditLog{}
	paginationClone := *pagination

	// Build query with filters
	queryBase := fmt.Sprintf("SELECT id, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at FROM %s", KeySpace+"."+schemas.Collections.AuditLog)
	countBase := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.AuditLog)

	// Every filter column below is backed by a secondary index (see
	// provider.go), so equality restrictions need no ALLOW FILTERING — Scylla
	// builds those indexes as materialized views and serves them directly.
	//
	// The created_at BOUNDS are different: a range on a non-primary-key column
	// cannot be served by an index, so it forces ALLOW FILTERING and a scan.
	// Only the timestamp bounds set that flag; an indexed-equality-only query
	// keeps its existing index-served plan.
	//
	// ponytail: ALLOW FILTERING for the timestamp range. This is an admin-only,
	// rarely-run query and the table is small relative to user data. The upgrade
	// path when it stops being cheap is a materialized view keyed on a coarse
	// time bucket, not a bigger scan.
	clauses := []string{}
	filterValues := []interface{}{}
	needsAllowFiltering := false

	addEq := func(col string, v interface{}) {
		clauses = append(clauses, col+"=?")
		filterValues = append(filterValues, v)
	}

	if action, ok := filter["action"]; ok && action != "" {
		addEq("action", action)
	}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		addEq("actor_id", actorID)
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		addEq("resource_type", resourceType)
	}
	if resourceID, ok := filter["resource_id"]; ok && resourceID != "" {
		addEq("resource_id", resourceID)
	}
	if fromTimestamp, ok := filter["from_timestamp"]; ok {
		clauses = append(clauses, "created_at>=?")
		filterValues = append(filterValues, fromTimestamp)
		needsAllowFiltering = true
	}
	if toTimestamp, ok := filter["to_timestamp"]; ok {
		clauses = append(clauses, "created_at<=?")
		filterValues = append(filterValues, toTimestamp)
		needsAllowFiltering = true
	}

	whereClause := ""
	if len(clauses) > 0 {
		whereClause = " WHERE " + strings.Join(clauses, " AND ")
	}
	// More than one restriction on non-primary-key columns cannot be served by a
	// single index either, so Cassandra/Scylla requires the scan hint there too.
	if len(clauses) > 1 {
		needsAllowFiltering = true
	}
	if needsAllowFiltering {
		whereClause += " ALLOW FILTERING"
	}

	countQuery := countBase + whereClause
	err := p.db.Query(countQuery, filterValues...).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch with pagination
	// CQL grammar: LIMIT precedes ALLOW FILTERING, so the hint cannot simply be
	// carried along on whereClause here.
	query := queryBase + strings.TrimSuffix(whereClause, " ALLOW FILTERING") +
		fmt.Sprintf(" LIMIT %d", pagination.Limit+pagination.Offset)
	if needsAllowFiltering {
		query += " ALLOW FILTERING"
	}
	scanner := p.db.Query(query, filterValues...).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var auditLog schemas.AuditLog
			err := scanner.Scan(
				&auditLog.ID, &auditLog.ActorID, &auditLog.ActorType,
				&auditLog.ActorEmail, &auditLog.Action, &auditLog.ResourceType, &auditLog.ResourceID,
				&auditLog.IPAddress, &auditLog.UserAgent, &auditLog.Metadata,
				&auditLog.CreatedAt)
			if err != nil {
				return nil, nil, err
			}
			auditLogs = append(auditLogs, &auditLog)
		}
		counter++
	}

	return auditLogs, &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	// Partition key is id only; range scans on created_at require ALLOW FILTERING.
	// Secondary indexes support equality on created_at, not created_at < ?.
	query := fmt.Sprintf("SELECT id FROM %s WHERE created_at < ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.AuditLog)
	scanner := p.db.Query(query, before).Iter().Scanner()
	for scanner.Next() {
		var id string
		if err := scanner.Scan(&id); err != nil {
			return err
		}
		deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.AuditLog)
		if err := p.db.Query(deleteQuery, id).Exec(); err != nil {
			return err
		}
	}
	return scanner.Err()
}
