package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// MFA session methods implement the database-backed memory store for MFA sessions.
// Used when Redis is not configured; the memory_store/db provider delegates to these.
// Table/collection: schemas.Collections.MFASession ("authorizer_mfa_sessions")
// Key fields: user_id, key_name (maps to "key" param in Get/Delete), expires_at

// AddMFASession adds an MFA session to the database.
// Session fields: ID, UserID, KeyName, ExpiresAt, CreatedAt, UpdatedAt
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
	// TODO: insert session into schemas.Collections.MFASession
	return nil
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key (KeyName).
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	// TODO: query where user_id = ? AND key_name = ?
	var session *schemas.MFASession
	return session, nil
}

// DeleteMFASession deletes an MFA session by ID.
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	// TODO: delete where id = ?
	return nil
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key (KeyName).
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	// TODO: delete where user_id = ? AND key_name = ?
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID.
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	// TODO: query where user_id = ?
	var sessions []*schemas.MFASession
	return sessions, nil
}

// CleanExpiredMFASessions removes expired MFA sessions (expires_at < now).
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	// TODO: delete where expires_at < current_unix_timestamp
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing).
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	// TODO: select all from schemas.Collections.MFASession
	var sessions []*schemas.MFASession
	return sessions, nil
}
