package sql

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddSessionToken adds a session token to the database
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
	return p.db.Create(token).Error
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	err := p.db.Where("user_id = ? AND key_name = ?", userId, key).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	return p.db.Where("id = ?", id).Delete(&schemas.SessionToken{}).Error
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	return p.db.Where("user_id = ? AND key_name = ?", userId, key).Delete(&schemas.SessionToken{}).Error
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
// The userId parameter can be just the user ID (e.g., "123") or the full format (e.g., "auth_provider:123")
// This matches the in-memory store behavior which uses strings.Contains
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	// Match user_id that contains the userId string anywhere
	return p.db.Where("user_id LIKE ?", "%"+userId+"%").Delete(&schemas.SessionToken{}).Error
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace (e.g., "auth_provider")
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return p.db.Where("user_id LIKE ?", namespace+":%").Delete(&schemas.SessionToken{}).Error
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	return p.db.Where("expires_at < ?", currentTime).Delete(&schemas.SessionToken{}).Error
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	err := p.db.Find(&tokens).Error
	return tokens, err
}
