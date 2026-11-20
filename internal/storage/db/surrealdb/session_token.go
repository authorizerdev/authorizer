package surrealdb

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
	return p.create(ctx, schemas.Collections.SessionToken, token.ID, token)
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	return getWhere[schemas.SessionToken](ctx, p, schemas.Collections.SessionToken, "user_id = $u AND key_name = $k", "session token", map[string]any{"u": userId, "k": key})
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	return p.deleteByID(ctx, schemas.Collections.SessionToken, id)
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key,
// and reports whether this call deleted it. The delete is one statement, so
// under concurrent calls exactly one sees the row: that caller won.
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (bool, error) {
	return p.deleteReportingWinner(ctx, schemas.Collections.SessionToken, "user_id = $u AND key_name = $k", map[string]any{"u": userId, "k": key})
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID.
//
// The match is anchored, as on every backend: the exact value, or a
// ":<userId>" suffix. Stored user_id is "[<loginMethod>:]<userID>"
// (memory_store SetUserSession), so the user id is always the final segment.
// string::ends_with matches literally: no pattern syntax to escape.
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE user_id = $u OR string::ends_with(user_id, $suffix) RETURN NONE;",
		map[string]any{"tb": schemas.Collections.SessionToken, "u": userId, "suffix": ":" + userId})
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE string::starts_with(user_id, $prefix) RETURN NONE;",
		map[string]any{"tb": schemas.Collections.SessionToken, "prefix": namespace + ":"})
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE expires_at < $now RETURN NONE;",
		map[string]any{"tb": schemas.Collections.SessionToken, "now": time.Now().Unix()})
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	rows, err := listWhere[schemas.SessionToken](ctx, p, schemas.Collections.SessionToken, "", nil)
	return ptrs(rows), err
}

// deleteReportingWinner deletes the records of table matching where and
// reports whether any were deleted by this call.
func (p *provider) deleteReportingWinner(ctx context.Context, table, where string, vars map[string]any) (bool, error) {
	vars["tb"] = table
	// RETURN BEFORE: a projection would be evaluated on the deleted (NONE)
	// record. Only the row count matters here.
	rows, err := queryAll[map[string]any](ctx, p, "DELETE type::table($tb) WHERE "+where+" RETURN BEFORE;", vars)
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}
