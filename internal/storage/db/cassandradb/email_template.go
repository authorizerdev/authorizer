package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddEmailTemplate to add EmailTemplate
func (p *provider) AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	if emailTemplate.ID == "" {
		emailTemplate.ID = uuid.New().String()
	}
	emailTemplate.Key = emailTemplate.ID
	emailTemplate.CreatedAt = time.Now().Unix()
	emailTemplate.UpdatedAt = time.Now().Unix()
	existingEmailTemplate, _ := p.GetEmailTemplateByEventName(ctx, emailTemplate.EventName)
	if existingEmailTemplate != nil {
		return nil, fmt.Errorf("email template with %s event_name already exists", emailTemplate.EventName)
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, event_name, subject, design, template, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.EmailTemplate)
	err := p.db.Query(insertQuery, emailTemplate.ID, emailTemplate.EventName, emailTemplate.Subject, emailTemplate.Design, emailTemplate.Template, emailTemplate.CreatedAt, emailTemplate.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	emailTemplate.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(emailTemplate)
	if err != nil {
		return nil, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	emailTemplateMap := map[string]interface{}{}
	err = decoder.Decode(&emailTemplateMap)
	if err != nil {
		return nil, err
	}
	convertMapValues(emailTemplateMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range emailTemplateMap {
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
		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	updateValues = append(updateValues, emailTemplate.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.EmailTemplate, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}

	return emailTemplate, nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error) {
	emailTemplates := []*schemas.EmailTemplate{}
	paginationClone := pagination

	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.EmailTemplate)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT id, event_name, subject, design, template, created_at, updated_at FROM %s LIMIT %d", KeySpace+"."+schemas.Collections.EmailTemplate, pagination.Limit+pagination.Offset)

	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var emailTemplate schemas.EmailTemplate
			err := scanner.Scan(&emailTemplate.ID, &emailTemplate.EventName, &emailTemplate.Subject, &emailTemplate.Design, &emailTemplate.Template, &emailTemplate.CreatedAt, &emailTemplate.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			emailTemplates = append(emailTemplates, &emailTemplate)
		}
		counter++
	}
	return emailTemplates, paginationClone, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error) {
	var emailTemplate schemas.EmailTemplate
	query := fmt.Sprintf(`SELECT id, event_name, subject, design, template, created_at, updated_at FROM %s WHERE id = ? LIMIT 1`, KeySpace+"."+schemas.Collections.EmailTemplate)
	err := p.db.Query(query, emailTemplateID).Consistency(gocql.One).Scan(&emailTemplate.ID, &emailTemplate.EventName, &emailTemplate.Subject, &emailTemplate.Design, &emailTemplate.Template, &emailTemplate.CreatedAt, &emailTemplate.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &emailTemplate, nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error) {
	var emailTemplate schemas.EmailTemplate
	query := fmt.Sprintf(`SELECT id, event_name, subject, design, template, created_at, updated_at FROM %s WHERE event_name = ? LIMIT 1 ALLOW FILTERING`, KeySpace+"."+schemas.Collections.EmailTemplate)
	err := p.db.Query(query, eventName).Consistency(gocql.One).Scan(&emailTemplate.ID, &emailTemplate.EventName, &emailTemplate.Subject, &emailTemplate.Design, &emailTemplate.Template, &emailTemplate.CreatedAt, &emailTemplate.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &emailTemplate, nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.EmailTemplate)
	err := p.db.Query(query, emailTemplate.ID).Exec()
	if err != nil {
		return err
	}

	return nil
}
