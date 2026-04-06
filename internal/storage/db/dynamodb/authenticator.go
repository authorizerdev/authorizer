package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func (p *provider) AddAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	exists, _ := p.GetAuthenticatorDetailsByUserId(ctx, authenticators.UserID, authenticators.Method)
	if exists != nil {
		return authenticators, nil
	}
	if authenticators.ID == "" {
		authenticators.ID = uuid.New().String()
	}
	authenticators.CreatedAt = time.Now().Unix()
	authenticators.UpdatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.Authenticators, authenticators); err != nil {
		return nil, err
	}
	return authenticators, nil
}

func (p *provider) UpdateAuthenticator(ctx context.Context, authenticators *schemas.Authenticator) (*schemas.Authenticator, error) {
	if authenticators.ID != "" {
		authenticators.UpdatedAt = time.Now().Unix()
		if err := p.updateByHashKey(ctx, schemas.Collections.Authenticators, "id", authenticators.ID, authenticators); err != nil {
			return nil, err
		}
	}
	return authenticators, nil
}

func (p *provider) GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error) {
	f := expression.Name("user_id").Equal(expression.Value(userId)).And(expression.Name("method").Equal(expression.Value(authenticatorType)))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.Authenticators, nil, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	var a schemas.Authenticator
	if err := unmarshalItem(items[0], &a); err != nil {
		return nil, err
	}
	return &a, nil
}
