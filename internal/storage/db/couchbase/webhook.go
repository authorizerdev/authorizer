package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebhook to add webhook
func (p *provider) AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
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
	_, err := p.db.Collection(schemas.Collections.Webhook).Insert(webhook.ID, webhook, &insertOpt)
	if err != nil {
		return nil, err
	}
	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error) {
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
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id='%s'`, p.scopeName, schemas.Collections.Webhook, updateFields, webhook.ID)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}

	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error) {
	webhooks := []*schemas.Webhook{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Webhook)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s.%s OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.Webhook)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var webhook schemas.Webhook
		err := queryResult.Row(&webhook)
		if err != nil {
			log.Fatal(err)
		}
		webhooks = append(webhooks, &webhook)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return webhooks, paginationClone, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error) {
	var webhook *schemas.Webhook
	params := make(map[string]interface{}, 1)
	params["_id"] = webhookID
	query := fmt.Sprintf(`SELECT _id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, schemas.Collections.Webhook)
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
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error) {
	params := make(map[string]interface{}, 1)
	// params["event_name"] = eventName + "%"
	query := fmt.Sprintf(`SELECT _id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s.%s WHERE event_name LIKE '%s'`, p.scopeName, schemas.Collections.Webhook, eventName+"%")
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	webhooks := []*schemas.Webhook{}
	for queryResult.Next() {
		var webhook *schemas.Webhook
		err := queryResult.Row(&webhook)
		if err != nil {
			log.Fatal(err)
		}
		webhooks = append(webhooks, webhook)
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return webhooks, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error {
	params := make(map[string]interface{}, 1)
	params["webhook_id"] = webhook.ID
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Webhook).Remove(webhook.ID, &removeOpt)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE webhook_id=$webhook_id`, p.scopeName, schemas.Collections.WebhookLog)
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
