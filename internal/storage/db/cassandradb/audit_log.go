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

	insertQuery := fmt.Sprintf("INSERT INTO %s (id, timestamp, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, organization_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.AuditLog)
	err := p.db.Query(insertQuery,
		auditLog.ID, auditLog.Timestamp, auditLog.ActorID, auditLog.ActorType, auditLog.ActorEmail,
		auditLog.Action, auditLog.ResourceType, auditLog.ResourceID, auditLog.IPAddress,
		auditLog.UserAgent, auditLog.Metadata, auditLog.OrganizationID,
		auditLog.CreatedAt, auditLog.UpdatedAt).Exec()
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
	queryBase := fmt.Sprintf("SELECT id, timestamp, actor_id, actor_type, actor_email, action, resource_type, resource_id, ip_address, user_agent, metadata, organization_id, created_at, updated_at FROM %s", KeySpace+"."+schemas.Collections.AuditLog)
	countBase := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.AuditLog)

	whereClause := ""
	filterValues := []interface{}{}

	if action, ok := filter["action"]; ok && action != "" {
		whereClause += " WHERE action=?"
		filterValues = append(filterValues, action)
	}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		if whereClause == "" {
			whereClause += " WHERE actor_id=?"
		} else {
			whereClause += " AND actor_id=?"
		}
		filterValues = append(filterValues, actorID)
	}

	allowFiltering := ""
	if whereClause != "" {
		allowFiltering = " ALLOW FILTERING"
	}

	// Count total
	countQuery := countBase + whereClause + allowFiltering
	err := p.db.Query(countQuery, filterValues...).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// Fetch with pagination
	query := queryBase + whereClause + fmt.Sprintf(" LIMIT %d", pagination.Limit+pagination.Offset) + allowFiltering
	scanner := p.db.Query(query, filterValues...).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var auditLog schemas.AuditLog
			err := scanner.Scan(
				&auditLog.ID, &auditLog.Timestamp, &auditLog.ActorID, &auditLog.ActorType,
				&auditLog.ActorEmail, &auditLog.Action, &auditLog.ResourceType, &auditLog.ResourceID,
				&auditLog.IPAddress, &auditLog.UserAgent, &auditLog.Metadata, &auditLog.OrganizationID,
				&auditLog.CreatedAt, &auditLog.UpdatedAt)
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
	// Cassandra doesn't support range deletes without knowing the partition key
	// So we need to first fetch IDs, then delete them
	query := fmt.Sprintf("SELECT id FROM %s WHERE timestamp < ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.AuditLog)
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
	return nil
}
