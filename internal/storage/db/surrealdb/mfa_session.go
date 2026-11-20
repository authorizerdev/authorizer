package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddMFASession adds an MFA session to the database
func (p *provider) AddMFASession(ctx context.Context, session *schemas.MFASession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.Key = session.ID
	if session.CreatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = time.Now().Unix()
	}
	return p.create(ctx, schemas.Collections.MFASession, session.ID, session)
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	return getWhere[schemas.MFASession](ctx, p, schemas.Collections.MFASession, "user_id = $u AND key_name = $k", "mfa session", map[string]any{"u": userId, "k": key})
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	return p.deleteByID(ctx, schemas.Collections.MFASession, id)
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE user_id = $u AND key_name = $k RETURN NONE;",
		map[string]any{"tb": schemas.Collections.MFASession, "u": userId, "k": key})
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	rows, err := listWhere[schemas.MFASession](ctx, p, schemas.Collections.MFASession, "user_id = $u", map[string]any{"u": userId})
	return ptrs(rows), err
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE expires_at < $now RETURN NONE;",
		map[string]any{"tb": schemas.Collections.MFASession, "now": time.Now().Unix()})
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	rows, err := listWhere[schemas.MFASession](ctx, p, schemas.Collections.MFASession, "", nil)
	return ptrs(rows), err
}
