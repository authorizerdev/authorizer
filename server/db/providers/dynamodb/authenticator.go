package dynamodb

import (
	"context"
	"errors"
	"github.com/guregu/dynamo"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/server/db/models"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	collection := p.db.Table(models.Collections.Authenticators)
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}

	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	err := collection.Put(authenticators).RunWithContext(ctx)
	if err != nil {
		return &authenticators, err
	}
	return &authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	collection := p.db.Table(models.Collections.Authenticators)
	if authenticators.ID != "" {
		authenticators.UpdatedAt = time.Now().Unix()
		err := UpdateByHashKey(collection, "id", authenticators.ID, authenticators)
		if err != nil {
			return &authenticators, err
		}
	}
	return &authenticators, nil

}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticators, error) {
	collection := p.db.Table(models.Collections.Authenticators)
	var authenticators *models.Authenticators
	err := collection.Get("user_id", userId).Range("method", dynamo.Equal, authenticatorType).OneWithContext(ctx, &authenticators)
	if err != nil {
		if authenticators.ID == "" {
			return authenticators, errors.New("no documets found")
		} else {
			return authenticators, nil
		}
	}
	return authenticators, nil
}
