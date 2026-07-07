package sql

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// likeEscaper escapes LIKE metacharacters so a value is matched literally when
// used with an explicit ESCAPE '\' clause.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

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

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID.
// The userId parameter can be just the user ID (e.g., "123") or the full format
// (e.g., "auth_provider:123"). Stored user_id values are "[namespace:]<userID>"
// (see memory_store SetUserSession), so we match the exact value OR any
// ":<userId>" suffix. This is anchored (a bare substring match let "42" delete
// tokens for "142" or "provider:427") and metacharacters are escaped so "%"/"_"
// in an id are treated literally.
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	escaped := escapeLike(userId)
	return p.db.Where(`user_id = ? OR user_id LIKE ? ESCAPE '\'`, userId, "%:"+escaped).Delete(&schemas.SessionToken{}).Error
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace (e.g., "auth_provider").
// This is a legitimate prefix match: stored user_id is "<namespace>:<userID>".
// The namespace is escaped so its metacharacters (auth-recipe names contain "_")
// are matched literally rather than as LIKE wildcards.
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return p.db.Where(`user_id LIKE ? ESCAPE '\'`, escapeLike(namespace)+":%").Delete(&schemas.SessionToken{}).Error
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
