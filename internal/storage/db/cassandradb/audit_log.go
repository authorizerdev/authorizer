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
// auditLogIndexedColumns is the set of audit_log columns backed by a secondary
// index in NewProvider. The ALLOW FILTERING decision in buildAuditLogWhere is
// derived from this set rather than from the clause count alone, so adding or
// dropping an index is a one-line change here and cannot leave a hint-less
// equality against an unindexed column — which Cassandra rejects outright
// rather than merely running slowly.
//
// created_at is deliberately absent. Its index was dropped: a Cassandra
// secondary index serves equality only, and the only queries this table ever
// issues against created_at are the range bounds below and the range in
// DeleteAuditLogsBefore, so the index could never be used while still costing a
// write on every audit row — and audit_log is the highest-write table here.
var auditLogIndexedColumns = map[string]bool{
	"actor_id":      true,
	"action":        true,
	"resource_type": true,
	"resource_id":   true,
}

// buildAuditLogWhere builds the WHERE clause and bind values for ListAuditLogs,
// and reports whether the query needs ALLOW FILTERING.
//
// The hint is returned separately rather than appended to the clause because CQL
// grammar puts LIMIT before ALLOW FILTERING, so the paged query cannot simply
// carry the clause through.
//
// Three things force the hint, and the set above is what decides the first two:
//
//  1. An equality on a column with no secondary index.
//  2. A range restriction. A range on a non-primary-key column cannot be served
//     by a secondary index even when the column has one, so this holds
//     regardless of auditLogIndexedColumns.
//  3. More than one restriction. A single query can only be served by one
//     index, so Cassandra/Scylla requires the hint for any second clause.
//
// ponytail: (3) is the common case, not the rare one. "this actor, this action"
// is an ordinary admin search and it scans. That is the right trade for an
// admin-only paginated endpoint — an erroring filter combination is worse than a
// slow one, and before #802 that combination errored outright on Scylla. The
// upgrade path when it stops being cheap is a composite materialized view over
// the filter combinations that actually get used, not a bigger scan.
func buildAuditLogWhere(filter map[string]interface{}) (string, []interface{}, bool) {
	clauses := []string{}
	values := []interface{}{}
	needsAllowFiltering := false

	addEq := func(col string, v interface{}) {
		clauses = append(clauses, col+"=?")
		values = append(values, v)
		if !auditLogIndexedColumns[col] {
			needsAllowFiltering = true
		}
	}
	addRange := func(expr string, v interface{}) {
		clauses = append(clauses, expr)
		values = append(values, v)
		needsAllowFiltering = true
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
		addRange("created_at>=?", fromTimestamp)
	}
	if toTimestamp, ok := filter["to_timestamp"]; ok {
		addRange("created_at<=?", toTimestamp)
	}

	if len(clauses) == 0 {
		return "", values, false
	}
	if len(clauses) > 1 {
		needsAllowFiltering = true
	}
	return " WHERE " + strings.Join(clauses, " AND "), values, needsAllowFiltering
}

func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	auditLogs := []*schemas.AuditLog{}
	paginationClone := *pagination

	// Build query with filters
	queryBase := fmt.Sprintf("SELECT id, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at FROM %s", KeySpace+"."+schemas.Collections.AuditLog)
	countBase := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.AuditLog)

	whereClause, filterValues, needsAllowFiltering := buildAuditLogWhere(filter)

	countQuery := countBase + whereClause
	if needsAllowFiltering {
		countQuery += " ALLOW FILTERING"
	}
	err := p.db.Query(countQuery, filterValues...).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch with pagination. CQL grammar puts LIMIT before ALLOW FILTERING, which
	// is why buildAuditLogWhere returns the hint as a flag rather than baked into
	// the clause.
	query := queryBase + whereClause + fmt.Sprintf(" LIMIT %d", pagination.Limit+pagination.Offset)
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
	// A scan that dies part-way — read timeout, coordinator failure — ends
	// Next() normally, so without this the call returns a TRUNCATED page with a
	// nil error while paginationClone.Total (a separate query) reports the real
	// count. An admin would see an incomplete audit trail with no signal it was
	// cut short, which is the exact failure this table exists to prevent. Newly
	// reachable here: the filters above can now produce ALLOW FILTERING scans,
	// where a partial read is far likelier than on an index-served equality.
	// DeleteAuditLogsBefore below already does this.
	if err := scanner.Err(); err != nil {
		return nil, nil, err
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
