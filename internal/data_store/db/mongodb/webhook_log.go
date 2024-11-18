package mongodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddWebhookLog to add webhook log
func (p *provider) AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*model.WebhookLog, error) {
	if webhookLog.ID == "" {
		webhookLog.ID = uuid.New().String()
	}

	webhookLog.Key = webhookLog.ID
	webhookLog.CreatedAt = time.Now().Unix()
	webhookLog.UpdatedAt = time.Now().Unix()

	webhookLogCollection := p.db.Collection(schemas.Collections.WebhookLog, options.Collection())
	_, err := webhookLogCollection.InsertOne(ctx, webhookLog)
	if err != nil {
		return nil, err
	}
	return webhookLog.AsAPIWebhookLog(), nil
}

// ListWebhookLogs to list webhook logs
func (p *provider) ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) (*model.WebhookLogs, error) {
	webhookLogs := []*model.WebhookLog{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})

	paginationClone := pagination
	query := bson.M{}

	if webhookID != "" {
		query = bson.M{"webhook_id": webhookID}
	}

	webhookLogCollection := p.db.Collection(schemas.Collections.WebhookLog, options.Collection())
	count, err := webhookLogCollection.CountDocuments(ctx, query, options.Count())
	if err != nil {
		return nil, err
	}

	paginationClone.Total = count

	cursor, err := webhookLogCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var webhookLog *schemas.WebhookLog
		err := cursor.Decode(&webhookLog)
		if err != nil {
			return nil, err
		}
		webhookLogs = append(webhookLogs, webhookLog.AsAPIWebhookLog())
	}

	return &model.WebhookLogs{
		Pagination:  paginationClone,
		WebhookLogs: webhookLogs,
	}, nil
}
