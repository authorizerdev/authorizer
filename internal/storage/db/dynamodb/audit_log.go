package dynamodb

import (
	"context"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	return p.putItem(ctx, schemas.Collections.AuditLog, auditLog)
}

func int64FromFilter(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case int32:
		return int64(x), true
	case float64:
		return int64(x), true
	default:
		return 0, false
	}
}

// auditLogExtraFilter builds a filter for attributes not covered by the primary key condition
// (omitKey is "action" or "actor_id" when that attribute is the partition key for the Query).
func auditLogExtraFilter(filter map[string]interface{}, omitKey string) *expression.ConditionBuilder {
	var conds []expression.ConditionBuilder
	if omitKey != "actor_id" {
		if actorID, ok := filter["actor_id"]; ok && actorID != "" {
			conds = append(conds, expression.Name("actor_id").Equal(expression.Value(actorID)))
		}
	}
	if omitKey != "action" {
		if action, ok := filter["action"]; ok && action != "" {
			conds = append(conds, expression.Name("action").Equal(expression.Value(action)))
		}
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		conds = append(conds, expression.Name("resource_type").Equal(expression.Value(resourceType)))
	}
	if resourceID, ok := filter["resource_id"]; ok && resourceID != "" {
		conds = append(conds, expression.Name("resource_id").Equal(expression.Value(resourceID)))
	}
	if v, ok := filter["from_timestamp"]; ok {
		if ts, ok := int64FromFilter(v); ok {
			conds = append(conds, expression.Name("created_at").GreaterThanEqual(expression.Value(ts)))
		}
	}
	if v, ok := filter["to_timestamp"]; ok {
		if ts, ok := int64FromFilter(v); ok {
			conds = append(conds, expression.Name("created_at").LessThanEqual(expression.Value(ts)))
		}
	}
	if len(conds) == 0 {
		return nil
	}
	merged := conds[0]
	for i := 1; i < len(conds); i++ {
		merged = merged.And(conds[i])
	}
	return &merged
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	paginationClone := *pagination

	var actionVal, actorVal string
	if a, ok := filter["action"]; ok && a != "" {
		if s, ok := a.(string); ok {
			actionVal = s
		}
	}
	if a, ok := filter["actor_id"]; ok && a != "" {
		if s, ok := a.(string); ok {
			actorVal = s
		}
	}

	var items []map[string]types.AttributeValue
	var err error
	table := schemas.Collections.AuditLog

	switch {
	case actionVal != "":
		extra := auditLogExtraFilter(filter, "action")
		items, err = p.queryEq(ctx, table, "action", "action", actionVal, extra)
	case actorVal != "":
		extra := auditLogExtraFilter(filter, "actor_id")
		items, err = p.queryEq(ctx, table, "actor_id", "actor_id", actorVal, extra)
	default:
		extra := auditLogExtraFilter(filter, "")
		items, err = p.scanAllRaw(ctx, table, nil, extra)
	}
	if err != nil {
		return nil, nil, err
	}

	var logs []*schemas.AuditLog
	for _, it := range items {
		var a schemas.AuditLog
		if err := unmarshalItem(it, &a); err != nil {
			return nil, nil, err
		}
		logs = append(logs, &a)
	}

	sort.Slice(logs, func(i, j int) bool { return logs[i].CreatedAt > logs[j].CreatedAt })

	total := int64(len(logs))
	paginationClone.Total = total

	start := int(pagination.Offset)
	if start >= len(logs) {
		return []*schemas.AuditLog{}, &paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(logs) {
		end = len(logs)
	}

	return logs[start:end], &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	f := expression.Name("created_at").LessThan(expression.Value(before))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.AuditLog, nil, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var a schemas.AuditLog
		if err := unmarshalItem(it, &a); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.AuditLog, "id", a.ID); err != nil {
			return err
		}
	}
	return nil
}
