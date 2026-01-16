package cassandradb

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Stub implementations - TODO: Implement for CassandraDB

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

func (p *provider) AddMFASession(ctx context.Context, session *schemas.MFASession) error {
	return nil
}

func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	return nil, nil
}

func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	return nil
}

func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	return nil
}

func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	return nil, nil
}

func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	return nil
}

func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	return nil, nil
}

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
