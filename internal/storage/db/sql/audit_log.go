package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

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
	res := p.db.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&auditLog)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	var auditLogs []*schemas.AuditLog
	var total int64

	query := p.db.Model(&schemas.AuditLog{})

	// Apply filters
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		query = query.Where("actor_id = ?", actorID)
	}
	if action, ok := filter["action"]; ok && action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if resourceID, ok := filter["resource_id"]; ok && resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}
	if fromTimestamp, ok := filter["from_timestamp"]; ok {
		query = query.Where("created_at >= ?", fromTimestamp)
	}
	if toTimestamp, ok := filter["to_timestamp"]; ok {
		query = query.Where("created_at <= ?", toTimestamp)
	}

	// Count total
	totalRes := query.Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}

	// Fetch results
	result := query.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&auditLogs)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	paginationClone := *pagination
	paginationClone.Total = total

	return auditLogs, &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	res := p.db.Where("created_at < ?", before).Delete(&schemas.AuditLog{})
	if res.Error != nil {
		return res.Error
	}
	return nil
}
