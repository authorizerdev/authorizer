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

// CreateApplication creates a new M2M application
func (p *provider) CreateApplication(ctx context.Context, application *schemas.Application) error {
	if application.ID == "" {
		application.ID = uuid.New().String()
	}
	application.Key = application.ID
	application.CreatedAt = time.Now().Unix()
	application.UpdatedAt = time.Now().Unix()
	applicationCollection := p.db.Collection(schemas.Collections.Application, options.Collection())
	_, err := applicationCollection.InsertOne(ctx, application)
	if err != nil {
		return err
	}
	return nil
}

// GetApplicationByID retrieves an application by ID
func (p *provider) GetApplicationByID(ctx context.Context, id string) (*schemas.Application, error) {
	var application schemas.Application
	applicationCollection := p.db.Collection(schemas.Collections.Application, options.Collection())
	err := applicationCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&application)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// GetApplicationByClientID retrieves an application by client ID
func (p *provider) GetApplicationByClientID(ctx context.Context, clientID string) (*schemas.Application, error) {
	var application schemas.Application
	applicationCollection := p.db.Collection(schemas.Collections.Application, options.Collection())
	err := applicationCollection.FindOne(ctx, bson.M{"client_id": clientID}).Decode(&application)
	if err != nil {
		return nil, err
	}
	return &application, nil
}

// ListApplications lists all applications with pagination
func (p *provider) ListApplications(ctx context.Context, pagination *model.Pagination) ([]*schemas.Application, *model.Pagination, error) {
	applications := []*schemas.Application{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	applicationCollection := p.db.Collection(schemas.Collections.Application, options.Collection())
	count, err := applicationCollection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := applicationCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var application schemas.Application
		err := cursor.Decode(&application)
		if err != nil {
			return nil, nil, err
		}
		applications = append(applications, &application)
	}
	return applications, paginationClone, nil
}

// UpdateApplication updates an application
func (p *provider) UpdateApplication(ctx context.Context, application *schemas.Application) error {
	application.UpdatedAt = time.Now().Unix()
	applicationCollection := p.db.Collection(schemas.Collections.Application, options.Collection())
	_, err := applicationCollection.ReplaceOne(ctx, bson.M{"_id": bson.M{"$eq": application.ID}}, application, options.Replace())
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication deletes an application by ID
func (p *provider) DeleteApplication(ctx context.Context, id string) error {
	applicationCollection := p.db.Collection(schemas.Collections.Application, options.Collection())
	_, err := applicationCollection.DeleteOne(ctx, bson.M{"_id": id}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}
