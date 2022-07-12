package mongodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()

	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	_, err := webhookCollection.InsertOne(ctx, webhook)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	_, err := webhookCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": webhook.ID}}, bson.M{"$set": webhook}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}

	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination model.Pagination) (*model.Webhooks, error) {
	var webhooks []*model.Webhook
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})

	paginationClone := pagination

	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	count, err := webhookCollection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, err
	}

	paginationClone.Total = count

	cursor, err := webhookCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var webhook models.Webhook
		err := cursor.Decode(&webhook)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook.AsAPIWebhook())
	}

	return &model.Webhooks{
		Pagination: &paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	var webhook models.Webhook
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	err := webhookCollection.FindOne(ctx, bson.M{"_id": webhookID}).Decode(&webhook)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) (*model.Webhook, error) {
	var webhook models.Webhook
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	err := webhookCollection.FindOne(ctx, bson.M{"event_name": eventName}).Decode(&webhook)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	_, err := webhookCollection.DeleteOne(nil, bson.M{"_id": webhook.ID}, options.Delete())
	if err != nil {
		return err
	}

	webhookLogCollection := p.db.Collection(models.Collections.WebhookLog, options.Collection())
	_, err = webhookLogCollection.DeleteMany(nil, bson.M{"webhook_id": webhook.ID}, options.Delete())
	if err != nil {
		return err
	}

	return nil
}
