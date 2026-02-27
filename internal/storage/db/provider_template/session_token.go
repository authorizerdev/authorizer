package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Session token methods implement the database-backed memory store for user sessions.
// Used when Redis is not configured; the memory_store/db provider delegates to these.
// Table/collection: schemas.Collections.SessionToken ("authorizer_session_tokens")
// Key fields: user_id, key_name (maps to "key" param in Get/Delete), expires_at

// AddSessionToken adds a session token to the database.
// Token fields: ID, UserID, KeyName (lookup key), Token, ExpiresAt, CreatedAt, UpdatedAt
func (p *provider) AddSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	token.Key = token.ID
	if token.CreatedAt == 0 {
		token.CreatedAt = time.Now().Unix()
	}
	if token.UpdatedAt == 0 {
		token.UpdatedAt = time.Now().Unix()
	}
	// TODO: insert token into schemas.Collections.SessionToken
	return nil
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key (KeyName).
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	// TODO: query where user_id = ? AND key_name = ?
	var token *schemas.SessionToken
	return token, nil
}

// DeleteSessionToken deletes a session token by ID.
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	// TODO: delete where id = ?
	return nil
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key (KeyName).
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	// TODO: delete where user_id = ? AND key_name = ?
	return nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID.
// The userId can be just the ID (e.g., "123") or full format (e.g., "auth_provider:123").
// Implement using LIKE/contains to match user_id containing userId.
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	// TODO: delete where user_id LIKE ? (e.g., "%userId%")
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace (e.g., "auth_provider").
// Namespace format: "namespace:". Match user_id that starts with "namespace:".
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	// TODO: delete where user_id LIKE ?
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens (expires_at < now).
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	// TODO: delete where expires_at < current_unix_timestamp
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing).
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	// TODO: select all from schemas.Collections.SessionToken
	var tokens []*schemas.SessionToken
	return tokens, nil
}
