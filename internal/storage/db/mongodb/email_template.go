package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	emailTemplateCollection := p.db.Collection(schemas.Collections.EmailTemplate, options.Collection())
	_, err := emailTemplateCollection.InsertOne(ctx, emailTemplate)
	if err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// UpdateEmailTemplate to update EmailTemplate
func (p *provider) UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error) {
	emailTemplate.UpdatedAt = time.Now().Unix()
	emailTemplateCollection := p.db.Collection(schemas.Collections.EmailTemplate, options.Collection())
	_, err := emailTemplateCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": emailTemplate.ID}}, bson.M{"$set": emailTemplate}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// ListEmailTemplates to list EmailTemplate
func (p *provider) ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error) {
	var emailTemplates []*schemas.EmailTemplate
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	emailTemplateCollection := p.db.Collection(schemas.Collections.EmailTemplate, options.Collection())
	count, err := emailTemplateCollection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := emailTemplateCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var emailTemplate *schemas.EmailTemplate
		err := cursor.Decode(&emailTemplate)
		if err != nil {
			return nil, nil, err
		}
		emailTemplates = append(emailTemplates, emailTemplate)
	}
	return emailTemplates, paginationClone, nil
}

// GetEmailTemplateByID to get EmailTemplate by id
func (p *provider) GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error) {
	var emailTemplate *schemas.EmailTemplate
	emailTemplateCollection := p.db.Collection(schemas.Collections.EmailTemplate, options.Collection())
	err := emailTemplateCollection.FindOne(ctx, bson.M{"_id": emailTemplateID}).Decode(&emailTemplate)
	if err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// GetEmailTemplateByEventName to get EmailTemplate by event_name
func (p *provider) GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error) {
	var emailTemplate *schemas.EmailTemplate
	emailTemplateCollection := p.db.Collection(schemas.Collections.EmailTemplate, options.Collection())
	err := emailTemplateCollection.FindOne(ctx, bson.M{"event_name": eventName}).Decode(&emailTemplate)
	if err != nil {
		return nil, err
	}
	return emailTemplate, nil
}

// DeleteEmailTemplate to delete EmailTemplate
func (p *provider) DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error {
	emailTemplateCollection := p.db.Collection(schemas.Collections.EmailTemplate, options.Collection())
	_, err := emailTemplateCollection.DeleteOne(ctx, bson.M{"_id": emailTemplate.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}
