package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/server/db/models"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return &authenticators, nil
	}

	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	authenticators.Key = authenticators.ID
	authenticatorsCollection := p.db.Collection(models.Collections.Authenticators, options.Collection())
	_, err := authenticatorsCollection.InsertOne(ctx, authenticators)
	if err != nil {
		return &authenticators, err
	}
	return &authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	authenticators.UpdatedAt = time.Now().Unix()
	authenticatorsCollection := p.db.Collection(models.Collections.Authenticators, options.Collection())
	_, err := authenticatorsCollection.UpdateOne(ctx, bson.D{{"_id", authenticators.ID}}, bson.M{"$set": authenticators})
	if err != nil {
		return &authenticators, err
	}
	return &authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticators, error) {
	var authenticators *models.Authenticators
	authenticatorsCollection := p.db.Collection(models.Collections.Authenticators, options.Collection())
	err := authenticatorsCollection.FindOne(ctx, bson.M{"user_id": userId, "method": authenticatorType}).Decode(&authenticators)
	if err != nil {
		return authenticators, err
	}
	return authenticators, nil
}
