package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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

	collection := p.db.Collection(schemas.Collections.AuditLog, options.Collection())
	_, err := collection.InsertOne(ctx, auditLog)
	if err != nil {
		return err
	}
	return nil
}

// ListAuditLogs queries audit logs with filters and pagination
func (p *provider) ListAuditLogs(ctx context.Context, pagination *model.Pagination, filter map[string]interface{}) ([]*schemas.AuditLog, *model.Pagination, error) {
	auditLogs := []*schemas.AuditLog{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"timestamp": -1})

	query := bson.M{}
	if actorID, ok := filter["actor_id"]; ok && actorID != "" {
		query["actor_id"] = actorID
	}
	if action, ok := filter["action"]; ok && action != "" {
		query["action"] = action
	}
	if resourceType, ok := filter["resource_type"]; ok && resourceType != "" {
		query["resource_type"] = resourceType
	}
	if resourceID, ok := filter["resource_id"]; ok && resourceID != "" {
		query["resource_id"] = resourceID
	}
	if orgID, ok := filter["organization_id"]; ok && orgID != "" {
		query["organization_id"] = orgID
	}
	if fromTimestamp, ok := filter["from_timestamp"]; ok {
		if query["timestamp"] == nil {
			query["timestamp"] = bson.M{}
		}
		query["timestamp"].(bson.M)["$gte"] = fromTimestamp
	}
	if toTimestamp, ok := filter["to_timestamp"]; ok {
		if query["timestamp"] == nil {
			query["timestamp"] = bson.M{}
		}
		query["timestamp"].(bson.M)["$lte"] = toTimestamp
	}

	paginationClone := *pagination
	collection := p.db.Collection(schemas.Collections.AuditLog, options.Collection())

	count, err := collection.CountDocuments(ctx, query, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count

	cursor, err := collection.Find(ctx, query, opts)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var auditLog *schemas.AuditLog
		err := cursor.Decode(&auditLog)
		if err != nil {
			return nil, nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, &paginationClone, nil
}

// DeleteAuditLogsBefore removes logs older than a timestamp
func (p *provider) DeleteAuditLogsBefore(ctx context.Context, before int64) error {
	collection := p.db.Collection(schemas.Collections.AuditLog, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"timestamp": bson.M{"$lt": before}})
	return err
}
