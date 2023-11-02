package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/gocql/gocql"
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
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, event_description, event_name, endpoint, headers, enabled,  created_at, updated_at) VALUES ('%s', '%s', '%s', '%s', '%s', %t, %d, %d)", KeySpace+"."+models.Collections.Webhook, webhook.ID, webhook.EventDescription, webhook.EventName, webhook.EndPoint, webhook.Headers, webhook.Enabled, webhook.CreatedAt, webhook.UpdatedAt)
	err := p.db.Query(insertQuery).Exec()
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
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = '%s'", KeySpace+"."+models.Collections.Webhook, updateFields, webhook.ID)
	err = p.db.Query(query).Exec()
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(ctx context.Context, pagination *model.Pagination) (*model.Webhooks, error) {
	webhooks := []*model.Webhook{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+models.Collections.Webhook)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, err
	}
	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s LIMIT %d", KeySpace+"."+models.Collections.Webhook, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var webhook models.Webhook
			err := scanner.Scan(&webhook.ID, &webhook.EventDescription, &webhook.EventName, &webhook.EndPoint, &webhook.Headers, &webhook.Enabled, &webhook.CreatedAt, &webhook.UpdatedAt)
			if err != nil {
				return nil, err
			}
			webhooks = append(webhooks, webhook.AsAPIWebhook())
		}
		counter++
	}

	return &model.Webhooks{
		Pagination: paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf(`SELECT id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s WHERE id = '%s' LIMIT 1`, KeySpace+"."+models.Collections.Webhook, webhookID)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&webhook.ID, &webhook.EventDescription, &webhook.EventName, &webhook.EndPoint, &webhook.Headers, &webhook.Enabled, &webhook.CreatedAt, &webhook.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return webhook.AsAPIWebhook(), nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(ctx context.Context, eventName string) ([]*model.Webhook, error) {
	query := fmt.Sprintf(`SELECT id, event_description, event_name, endpoint, headers, enabled, created_at, updated_at FROM %s WHERE event_name LIKE '%s' ALLOW FILTERING`, KeySpace+"."+models.Collections.Webhook, eventName+"%")
	scanner := p.db.Query(query).Iter().Scanner()
	webhooks := []*model.Webhook{}
	for scanner.Next() {
		var webhook models.Webhook
		err := scanner.Scan(&webhook.ID, &webhook.EventDescription, &webhook.EventName, &webhook.EndPoint, &webhook.Headers, &webhook.Enabled, &webhook.CreatedAt, &webhook.UpdatedAt)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook.AsAPIWebhook())
	}
	return webhooks, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(ctx context.Context, webhook *model.Webhook) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+models.Collections.Webhook, webhook.ID)
	err := p.db.Query(query).Exec()
	if err != nil {
		return err
	}

	getWebhookLogQuery := fmt.Sprintf("SELECT id FROM %s WHERE webhook_id = '%s' ALLOW FILTERING", KeySpace+"."+models.Collections.WebhookLog, webhook.ID)
	scanner := p.db.Query(getWebhookLogQuery).Iter().Scanner()
	webhookLogIDs := ""
	for scanner.Next() {
		var wlID string
		err = scanner.Scan(&wlID)
		if err != nil {
			return err
		}
		webhookLogIDs += fmt.Sprintf("'%s',", wlID)
	}
	webhookLogIDs = strings.TrimSuffix(webhookLogIDs, ",")
	query = fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+models.Collections.WebhookLog, webhookLogIDs)
	err = p.db.Query(query).Exec()
	return err
}
