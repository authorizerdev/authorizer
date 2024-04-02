package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook *models.Webhook) (*model.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}
	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()
	// Add timestamp to make event name unique for legacy version
	webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Webhook).Insert(webhook.ID, webhook, &insertOpt)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook *models.Webhook) (*model.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()
	// Event is changed
	if !strings.Contains(webhook.EventName, "-") {
		webhook.EventName = fmt.Sprintf("%s-%d", webhook.EventName, time.Now().Unix())
	}
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
	updateFields, params := GetSetFields(webhookMap)
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id='%s'`, p.scopeName, models.Collections.Webhook, updateFields, webhook.ID)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}

	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, models.Collections.Webhook)
	if err != nil {
		return nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s.%s OFFSET $offset LIMIT $limit", p.scopeName, models.Collections.Webhook)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
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
		Pagination: paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	var webhook *models.Webhook
	params := make(map[string]interface{}, 1)
	params["_id"] = webhookID
	query := fmt.Sprintf(`SELECT _id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, models.Collections.Webhook)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
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
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*model.Webhook, error) {
	params := make(map[string]interface{}, 1)
	// params["event_name"] = eventName + "%"
	query := fmt.Sprintf(`SELECT _id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s.%s WHERE event_name LIKE '%s'`, p.scopeName, models.Collections.Webhook, eventName+"%")
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	webhooks := []*model.Webhook{}
	for queryResult.Next() {
		var webhook *models.Webhook
		err := queryResult.Row(&webhook)
		if err != nil {
			log.Fatal(err)
		}
		webhooks = append(webhooks, webhook.AsAPIWebhook())
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return webhooks, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	params := make(map[string]interface{}, 1)
	params["webhook_id"] = webhook.ID
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Webhook).Remove(webhook.ID, &removeOpt)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE webhook_id=$webhook_id`, p.scopeName, models.Collections.WebhookLog)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	return nil
}
