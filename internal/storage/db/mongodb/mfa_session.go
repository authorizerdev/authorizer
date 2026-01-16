package mongodb

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Stub implementations for MongoDB provider
// TODO: Implement these methods for MongoDB

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
