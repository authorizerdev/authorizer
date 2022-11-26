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

// AddEmailTemplate to add EmailTemplate
func (p *provider) AddEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {

	if emailTemplate.ID == "" {
		emailTemplate.ID = uuid.New().String()
	}

	emailTemplate.Key = emailTemplate.ID
	emailTemplate.CreatedAt = time.Now().Unix()
	emailTemplate.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}

	_, err := p.db.Collection(models.Collections.EmailTemplate).Insert(emailTemplate.ID, emailTemplate, &insertOpt)
	if err != nil {
		return emailTemplate.AsAPIEmailTemplate(), err
	}

	return emailTemplate.AsAPIEmailTemplate(), nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate models.EmailTemplate) (*model.EmailTemplate, error) {
	scope := p.db.Scope("_default")
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

	updateFields := ""
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

		valueType := reflect.TypeOf(value)
		if valueType.Name() == "string" {
			updateFields += fmt.Sprintf("%s = '%s', ", key, value.(string))
		} else {
			updateFields += fmt.Sprintf("%s = %v, ", key, value)
		}
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")

	query := fmt.Sprintf("UPDATE auth._default.%s SET %s WHERE _id = '%s'", models.Collections.EmailTemplate, updateFields, emailTemplate.ID)
	_, err = scope.Query(query, &gocb.QueryOptions{})
	if err != nil {
		return nil, err
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination model.Pagination) (*model.EmailTemplates, error) {
	emailTemplates := []*model.EmailTemplate{}
	// r := p.db.Collection(models.Collections.User).
	paginationClone := pagination

	scope := p.db.Scope("_default")
	userQuery := fmt.Sprintf("SELECT  _id, event_name, subject, design, template, created_at, updated_at FROM auth._default.%s ORDER BY _id OFFSET %d LIMIT %d", models.Collections.EmailTemplate, paginationClone.Offset, paginationClone.Limit)

	queryResult, err := scope.Query(userQuery, &gocb.QueryOptions{
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
	})

	if err != nil {
		return nil, err
	}

	for queryResult.Next() {
		emailTemplate := models.EmailTemplate{}
		err := queryResult.Row(&emailTemplate)
		if err != nil {
			log.Fatal(err)
		}
		emailTemplates = append(emailTemplates, emailTemplate.AsAPIEmailTemplate())
	}

	if err := queryResult.Err(); err != nil {
		return nil, err

	}

	return &model.EmailTemplates{
		Pagination:     &paginationClone,
		EmailTemplates: emailTemplates,
	}, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*model.EmailTemplate, error) {
	emailTemplate := models.EmailTemplate{}
	time.Sleep(200 * time.Millisecond)

	scope := p.db.Scope("_default")
	query := fmt.Sprintf(`SELECT  _id, event_name, subject, design, template, created_at, updated_at  FROM auth._default.%s WHERE _id = '%s' LIMIT 1`, models.Collections.EmailTemplate, emailTemplateID)
	q, err := scope.Query(query, &gocb.QueryOptions{})

	if err != nil {
		return nil, err
	}
	err = q.One(&emailTemplate)

	if err != nil {
		return nil, err
	}

	return emailTemplate.AsAPIEmailTemplate(), nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*model.EmailTemplate, error) {
	emailTemplate := models.EmailTemplate{}
	time.Sleep(200 * time.Millisecond)

	scope := p.db.Scope("_default")
	query := fmt.Sprintf("SELECT  _id, event_name, subject, design, template, created_at, updated_at  FROM auth._default.%s WHERE event_name=$1 LIMIT 1", models.Collections.EmailTemplate)
	q, err := scope.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{eventName},
	})

	if err != nil {
		return nil, err
	}
	err = q.One(&emailTemplate)

	time.Sleep(20 * time.Second)
	if err != nil {
		return nil, err
	}

	return emailTemplate.AsAPIEmailTemplate(), nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *model.EmailTemplate) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.EmailTemplate).Remove(emailTemplate.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
