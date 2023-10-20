package dynamodb

import (
	"context"
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
	err := collection.Get().OneWithContext(ctx, &authenticators)
	if err != nil {
		if authenticators.Email == "" {
			return user, errors.New("no documets found")
		} else {
			return user, nil
		}
	}
	return user, nil
}
