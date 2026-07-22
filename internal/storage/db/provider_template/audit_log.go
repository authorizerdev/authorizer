package provider_template

import (
	"context"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddAuditLog adds an audit log entry
func (p *provider) AddAuditLog(ctx context.Context, log *schemas.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	return nil, nil, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp (retention)
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	return nil
}
