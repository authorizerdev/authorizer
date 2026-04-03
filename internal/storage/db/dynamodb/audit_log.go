package dynamodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddAuditLog adds an audit log entry
func (p *provider) AddAuditLog(ctx context.Context, auditLog *schemas.AuditLog) error {
	collection := p.db.Table(schemas.Collections.AuditLog)
	if auditLog.ID == "" {
		auditLog.ID = uuid.New().String()
	}
	auditLog.Key = auditLog.ID
	if auditLog.CreatedAt == 0 {
		auditLog.CreatedAt = time.Now().Unix()
	}
	err := collection.Put(auditLog).RunWithContext(ctx)
	if err != nil {
		return err
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	auditLogs := []*schemas.AuditLog{}
	var auditLog *schemas.AuditLog
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	var err error

	collection := p.db.Table(schemas.Collections.AuditLog)
	paginationClone := *pagination
	scanner := collection.Scan()

	// Apply filters
	if action, ok := filter["action"]; ok && action != "" {
		scanner = scanner.Filter("'action' = ?", action)
	}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		scanner = scanner.Filter("'actor_id' = ?", actorID)
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		scanner = scanner.Filter("'resource_type' = ?", resourceType)
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &auditLog) {
			if paginationClone.Offset == iteration {
				auditLogs = append(auditLogs, auditLog)
			}
		}
		err = iter.Err()
		if err != nil {
			return nil, nil, err
		}
		lastEval = iter.LastEvaluatedKey()
		iteration += paginationClone.Limit
	}

	// Count total matching documents
	var total int64
	countScanner := collection.Scan()
	if action, ok := filter["action"]; ok && action != "" {
		countScanner = countScanner.Filter("'action' = ?", action)
	}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		countScanner = countScanner.Filter("'actor_id' = ?", actorID)
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		countScanner = countScanner.Filter("'resource_type' = ?", resourceType)
	}
	var countItems []*schemas.AuditLog
	if err = countScanner.AllWithContext(ctx, &countItems); err != nil {
		return nil, nil, err
	}
	total = int64(len(countItems))
	paginationClone.Total = total

	return auditLogs, &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	collection := p.db.Table(schemas.Collections.AuditLog)
	var auditLogs []*schemas.AuditLog
	err := collection.Scan().Filter("'created_at' < ?", before).AllWithContext(ctx, &auditLogs)
	if err != nil {
		return err
	}
	for _, auditLog := range auditLogs {
		err := collection.Delete("id", auditLog.ID).RunWithContext(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
