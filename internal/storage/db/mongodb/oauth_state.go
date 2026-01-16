package mongodb

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Stub implementations for MongoDB provider
// TODO: Implement these methods for MongoDB

func (p *provider) AddOAuthState(ctx context.Context, state *schemas.OAuthState) error {
	return nil
}

func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	return nil, nil
}

func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	return nil
}

func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	return nil, nil
}
