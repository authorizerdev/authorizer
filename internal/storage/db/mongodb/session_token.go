package mongodb

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Stub implementations for MongoDB provider
// TODO: Implement these methods for MongoDB

func (p *provider) AddSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	return nil
}

func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	return nil, nil
}

func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	return nil
}

func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	return nil
}

func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	return nil
}

func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	return nil
}

func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	return nil, nil
}
