package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()

	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Webhook).Insert(webhook.ID, webhook, &insertOpt)
	if err != nil {
		return webhook.AsAPIWebhook(), err
	}
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	scope := p.db.Scope("_default")

	bytes, err := json.Marshal(webhook)
	if err != nil {
		return nil, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	webhookMap := map[string]interface{}{}
	err = decoder.Decode(&webhookMap)
	if err != nil {
		return nil, err
	}

	updateFields := ""
	for key, value := range webhookMap {
		if key == "_id" {
			continue
		}

		if key == "_key" {
			continue
		}

		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}

		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("%s = '%s', ", key, value.(string))
		} else {
			updateFields += fmt.Sprintf("%s = %v, ", key, value)
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	query := fmt.Sprintf("UPDATE auth._default.%s SET %s WHERE _id = '%s'", models.Collections.Webhook, updateFields, webhook.ID)
	_, err = scope.Query(query, &gocb.QueryOptions{})

	if err != nil {
		return nil, err
	}

	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}
	scope := p.db.Scope("_default")
	paginationClone := pagination

	query := fmt.Sprintf("SELECT _id, env, created_at, updated_at FROM auth._default.%s OFFSET %d LIMIT %d", models.Collections.Webhook, paginationClone.Offset, paginationClone.Limit)
	queryResult, err := scope.Query(query, &gocb.QueryOptions{})

	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var webhook models.Webhook
		err := queryResult.Row(&webhook)
		if err != nil {
			log.Fatal(err)
		}
		webhooks = append(webhooks, webhook.AsAPIWebhook())
	}

	if err := queryResult.Err(); err != nil {
		return nil, err

	}
	return &model.Webhooks{
		Pagination: &paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	var webhook models.Webhook
	scope := p.db.Scope("_default")
	query := fmt.Sprintf(`SELECT _id, event_name, endpoint, headers, enabled, created_at, updated_at FROM auth._default.%s WHERE _id = '%s' LIMIT 1`, models.Collections.Webhook, webhookID)
	q, err := scope.Query(query, &gocb.QueryOptions{})
	if err != nil {
		return nil, err
	}
	err = q.One(&webhook)

	if err != nil {
		return nil, err
	}

	return webhook.AsAPIWebhook(), nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) (*model.Webhook, error) {
	var webhook models.Webhook
	scope := p.db.Scope("_default")
	query := fmt.Sprintf(`SELECT _id, event_name, endpoint, headers, enabled, created_at, updated_at FROM auth._default.%s WHERE event_name = '%s' LIMIT 1`, models.Collections.Webhook, eventName)
	q, err := scope.Query(query, &gocb.QueryOptions{})

	if err != nil {
		return nil, err
	}
	err = q.One(&webhook)

	if err != nil {
		return nil, err
	}

	return webhook.AsAPIWebhook(), nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Webhook).Remove(webhook.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
