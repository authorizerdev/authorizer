package mongodb

import (
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(webhook models.Webhook) (models.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()

	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	_, err := webhookCollection.InsertOne(nil, webhook)
	if err != nil {
		return webhook, err
	}
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(webhook models.Webhook) (models.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	_, err := webhookCollection.UpdateOne(nil, bson.M{"_id": bson.M{"$eq": webhook.ID}}, bson.M{"$set": webhook}, options.MergeUpdateOptions())
	if err != nil {
		return webhook, err
	}

	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(pagination model.Pagination) (*model.Webhooks, error) {
	var webhooks []*model.Webhook
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})

	paginationClone := pagination

	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	count, err := webhookCollection.CountDocuments(nil, bson.M{}, options.Count())
	if err != nil {
		return nil, err
	}

	paginationClone.Total = count

	cursor, err := webhookCollection.Find(nil, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
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
func (p *provider) GetWebhookByID(webhookID string) (models.Webhook, error) {
	var webhook models.Webhook
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	err := webhookCollection.FindOne(nil, bson.M{"_id": webhookID}).Decode(&webhook)
	if err != nil {
		return webhook, err
	}
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(eventName string) (models.Webhook, error) {
	var webhook models.Webhook
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	err := webhookCollection.FindOne(nil, bson.M{"event_name": eventName}).Decode(&webhook)
	if err != nil {
		return webhook, err
	}
	return webhook, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(webhook models.Webhook) error {
	webhookCollection := p.db.Collection(models.Collections.Webhook, options.Collection())
	_, err := webhookCollection.DeleteOne(nil, bson.M{"_id": webhook.ID}, options.Delete())
	if err != nil {
		return err
	}

	return nil
}
