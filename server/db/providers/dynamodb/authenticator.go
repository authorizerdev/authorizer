package dynamodb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/server/db/models"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *models.Authenticator) (*models.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}

	collection := p.db.Table(models.Collections.Authenticators)
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}

	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	err := collection.Put(authenticators).RunWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *models.Authenticator) (*models.Authenticator, error) {
	collection := p.db.Table(models.Collections.Authenticators)
	if authenticators.ID != "" {
		authenticators.UpdatedAt = time.Now().Unix()
		err := UpdateByHashKey(collection, "id", authenticators.ID, authenticators)
		if err != nil {
			return nil, err
		}
	}
	return authenticators, nil

}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticator, error) {
	var authenticators *models.Authenticator
	collection := p.db.Table(models.Collections.Authenticators)
	iter := collection.Scan().Filter("'user_id' = ?", userId).Filter("'method' = ?", authenticatorType).Iter()
	for iter.NextWithContext(ctx, &authenticators) {
		return authenticators, nil
	}
	err := iter.Err()
	if err != nil {
		return nil, err
	}
	return authenticators, nil
}
