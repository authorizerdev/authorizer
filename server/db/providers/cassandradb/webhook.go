package cassandradb

import (
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
func (p *provider) AddWebhook(webhook models.Webhook) (models.Webhook, error) {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	webhook.Key = webhook.ID
	webhook.CreatedAt = time.Now().Unix()
	webhook.UpdatedAt = time.Now().Unix()

	insertQuery := fmt.Sprintf("INSERT INTO %s (id, event_name, endpoint, enabled,  created_at, updated_at) VALUES ('%s', '%s', '%s', %t, %d, %d)", KeySpace+"."+models.Collections.Webhook, webhook.ID, webhook.EventName, webhook.EndPoint, webhook.Enabled, webhook.CreatedAt, webhook.UpdatedAt)
	err := p.db.Query(insertQuery).Exec()
	if err != nil {
		return webhook, err
	}

	return webhook, nil
}

// UpdateWebhook to update webhook
func (p *provider) UpdateWebhook(webhook models.Webhook) (models.Webhook, error) {
	webhook.UpdatedAt = time.Now().Unix()

	bytes, err := json.Marshal(webhook)
	if err != nil {
		return webhook, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	webhookMap := map[string]interface{}{}
	err = decoder.Decode(&webhookMap)
	if err != nil {
		return webhook, err
	}

	updateFields := ""
	for key, value := range webhookMap {
		if key == "_id" {
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
		return webhook, err
	}
	return webhook, nil
}

// ListWebhooks to list webhook
func (p *provider) ListWebhook(pagination model.Pagination) (*model.Webhooks, error) {
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
	query := fmt.Sprintf("SELECT id, event_name, endpoint, enabled, created_at, updated_at FROM %s LIMIT %d", KeySpace+"."+models.Collections.Webhook, pagination.Limit+pagination.Offset)

	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var webhook models.Webhook
			err := scanner.Scan(&webhook.ID, &webhook.EventName, &webhook.EndPoint, &webhook.Enabled, &webhook.CreatedAt, &webhook.UpdatedAt)
			if err != nil {
				return nil, err
			}
			webhooks = append(webhooks, webhook.AsAPIWebhook())
		}
		counter++
	}

	return &model.Webhooks{
		Pagination: &paginationClone,
		Webhooks:   webhooks,
	}, nil
}

// GetWebhookByID to get webhook by id
func (p *provider) GetWebhookByID(webhookID string) (models.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf(`SELECT id, event_name, endpoint, enabled, created_at, updated_at FROM %s WHERE id = '%s' LIMIT 1`, KeySpace+"."+models.Collections.Webhook, webhookID)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&webhook.ID, &webhook.EventName, &webhook.EndPoint, &webhook.Enabled, &webhook.CreatedAt, &webhook.UpdatedAt)
	if err != nil {
		return webhook, err
	}
	return webhook, nil
}

// GetWebhookByEventName to get webhook by event_name
func (p *provider) GetWebhookByEventName(eventName string) (models.Webhook, error) {
	var webhook models.Webhook
	query := fmt.Sprintf(`SELECT id, event_name, endpoint, enabled, created_at, updated_at FROM %s WHERE event_name = '%s' LIMIT 1`, KeySpace+"."+models.Collections.Webhook, eventName)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&webhook.ID, &webhook.EventName, &webhook.EndPoint, &webhook.Enabled, &webhook.CreatedAt, &webhook.UpdatedAt)
	if err != nil {
		return webhook, err
	}
	return webhook, nil
}

// DeleteWebhook to delete webhook
func (p *provider) DeleteWebhook(webhook models.Webhook) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+models.Collections.Webhook, webhook.ID)
	err := p.db.Query(query).Exec()
	return err
}
