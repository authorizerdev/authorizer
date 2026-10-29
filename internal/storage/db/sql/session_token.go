package sql

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// likeEscapeChar is the ESCAPE character for every LIKE pattern built here.
//
// It is deliberately not a backslash. MySQL treats a backslash as an escape
// character inside string literals, so the literal `ESCAPE '\'` is unterminated
// there and the statement is rejected outright — `Error 1064 (42000): You have
// an error in your SQL syntax ... near ”\”` — while PostgreSQL and SQLite
// accept it. Session-token revocation ran through these two statements, so on
// MySQL it failed instead of revoking.
//
// Naming the escape character explicitly also makes a backslash in a value
// literal on every engine, which is why it is not escaped below: MySQL's LIKE
// otherwise treats backslash as an escape even with no ESCAPE clause.
const likeEscapeChar = "!"

// likeEscaper escapes LIKE metacharacters so a value is matched literally when
// used with an explicit ESCAPE clause.
var likeEscaper = strings.NewReplacer(`!`, `!!`, `%`, `!%`, `_`, `!_`)

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
	return p.db.WithContext(ctx).Create(token).Error
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	err := p.db.WithContext(ctx).Where("user_id = ? AND key_name = ?", userId, key).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	return p.db.WithContext(ctx).Where("id = ?", id).Delete(&schemas.SessionToken{}).Error
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (bool, error) {
	// One DELETE decides the race: the row lock serializes concurrent refresh
	// redemptions and RowsAffected tells the winner from the loser.
	res := p.db.WithContext(ctx).Where("user_id = ? AND key_name = ?", userId, key).Delete(&schemas.SessionToken{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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
	return p.db.WithContext(ctx).Where(`user_id = ? OR user_id LIKE ? ESCAPE '`+likeEscapeChar+`'`, userId, "%:"+escaped).Delete(&schemas.SessionToken{}).Error
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace (e.g., "auth_provider").
// This is a legitimate prefix match: stored user_id is "<namespace>:<userID>".
// The namespace is escaped so its metacharacters (auth-recipe names contain "_")
// are matched literally rather than as LIKE wildcards.
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return p.db.WithContext(ctx).Where(`user_id LIKE ? ESCAPE '`+likeEscapeChar+`'`, escapeLike(namespace)+":%").Delete(&schemas.SessionToken{}).Error
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	return p.db.WithContext(ctx).Where("expires_at < ?", currentTime).Delete(&schemas.SessionToken{}).Error
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	err := p.db.WithContext(ctx).Find(&tokens).Error
	return tokens, err
}
