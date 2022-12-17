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

	updateFields, params := GetSetFields(emailTemplateMap)
	params["emailId"] = emailTemplate.ID

	query := fmt.Sprintf("UPDATE %s.%s SET %s WHERE _id = $emailId", p.scopeName, models.Collections.EmailTemplate, updateFields)

	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return emailTemplate.AsAPIEmailTemplate(), nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination model.Pagination) (*model.EmailTemplates, error) {
	emailTemplates := []*model.EmailTemplate{}
	paginationClone := pagination

	_, paginationClone.Total = p.GetTotalDocs(ctx, models.Collections.EmailTemplate)

	userQuery := fmt.Sprintf("SELECT _id, event_name, subject, design, template, created_at, updated_at FROM %s.%s ORDER BY _id OFFSET $1 LIMIT $2", p.scopeName, models.Collections.EmailTemplate)

	queryResult, err := p.db.Query(userQuery, &gocb.QueryOptions{
		Context:              ctx,
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{paginationClone.Offset, paginationClone.Limit},
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

	query := fmt.Sprintf(`SELECT  _id, event_name, subject, design, template, created_at, updated_at  FROM %s.%s WHERE _id = $1 LIMIT 1`, p.scopeName, models.Collections.EmailTemplate)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{emailTemplateID},
	})

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

	query := fmt.Sprintf("SELECT  _id, event_name, subject, design, template, created_at, updated_at  FROM %s.%s WHERE event_name=$1 LIMIT 1", p.scopeName, models.Collections.EmailTemplate)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:              ctx,
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{eventName},
	})

	if err != nil {
		return nil, err
	}
	err = q.One(&emailTemplate)

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
