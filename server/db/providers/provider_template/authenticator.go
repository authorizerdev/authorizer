package provider_template

import (
	"context"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"time"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	return &authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators models.Authenticators) (*models.Authenticators, error) {
	authenticators.UpdatedAt = time.Now().Unix()
	return &authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticators, error) {
	var authenticators *models.Authenticators
	return authenticators, nil
}
