package surrealdb

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddAuditLog to add an audit log entry
func (p *provider) AddAuditLog(ctx context.Context, auditLog *schemas.AuditLog) error {
	if auditLog.ID == "" {
		auditLog.ID = uuid.New().String()
	}
	auditLog.Key = auditLog.ID
	if auditLog.CreatedAt == 0 {
		auditLog.CreatedAt = time.Now().Unix()
	}
	return p.create(ctx, schemas.Collections.AuditLog, auditLog.ID, auditLog)
}

// auditEqualityFilters are the filter keys matched by equality. Only these
// keys (and the two timestamps below) are read from the caller's filter map,
// each bound as a variable; any other key is ignored, as on every backend.
var auditEqualityFilters = []string{"actor_id", "action", "resource_type", "resource_id"}

// ListAuditLogs to list audit logs, newest first, filtered by the allow-listed keys
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	var conds []string
	vars := map[string]any{}
	for _, key := range auditEqualityFilters {
		if v, ok := filter[key]; ok && v != "" {
			conds = append(conds, key+" = $"+key)
			vars[key] = v
		}
	}
	if v, ok := filter["from_timestamp"]; ok {
		conds = append(conds, "created_at >= $from_timestamp")
		vars["from_timestamp"] = v
	}
	if v, ok := filter["to_timestamp"]; ok {
		conds = append(conds, "created_at <= $to_timestamp")
		vars["to_timestamp"] = v
	}
	rows, pag, err := page[schemas.AuditLog](ctx, p, schemas.Collections.AuditLog, strings.Join(conds, " AND "), vars, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE created_at < $cutoff RETURN NONE;",
		map[string]any{"tb": schemas.Collections.AuditLog, "cutoff": before})
}
