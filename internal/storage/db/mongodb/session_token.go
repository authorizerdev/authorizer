package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	_, err := collection.InsertOne(ctx, token)
	return err
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	err := collection.FindOne(ctx, bson.M{"user_id": userId, "key_name": key}).Decode(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (bool, error) {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	// DeleteOne, not DeleteMany: (user_id, key_name) identifies a single entry
	// and a single-document delete is atomic, so DeletedCount identifies the
	// caller that won the race.
	res, err := collection.DeleteOne(ctx, bson.M{"user_id": userId, "key_name": key})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID.
//
// The match is anchored: the exact value, or a ":<userId>" suffix. Stored
// user_id is "[<loginMethod>:]<userID>" (memory_store SetUserSession), so the
// user id is always the final segment.
//
// It used to be an unanchored substring match, which let a delete for "42"
// also take out tokens belonging to "142" or "provider:427". The SQL provider
// was anchored long ago; these backends kept the defect because the assertion
// covering it lived inside a SQL-only subtest. It now runs on every backend as
// "Anchored Session Token Delete".
//
// Three things are load-bearing in the pattern, and QuoteMeta is only one:
//
//   - regexp.QuoteMeta makes the NEEDLE literal. The old form interpolated
//     userId straight into $regex, so an id containing regex metacharacters
//     was evaluated as a pattern rather than matched literally.
//   - (?s) makes `.` cross a newline. Without it a stored "a\nb:<userId>" is
//     NOT matched, while every other backend deletes it — a missed revocation
//     on this backend only.
//   - \z, not $, anchors at true end-of-string. PCRE's $ also matches before a
//     trailing newline, so "<prefix>:<userId>\n" would be deleted here and
//     kept everywhere else — the over-delete primitive this change exists to
//     remove, surviving on one backend.
//
// None of the three is reachable from application writes today (login methods
// come from a fixed set, ids are UUIDv4), but "unreachable today" is exactly
// the standard the substring match was removed under.
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	pattern := "(?s)^(.*:)?" + regexp.QuoteMeta(userId) + `\z`
	_, err := collection.DeleteMany(ctx, bson.M{"user_id": bson.M{"$regex": pattern}})
	return err
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	// Escaped and (?s)-anchored for the same reasons as
	// DeleteAllSessionTokensByUserID above: the namespace reached $regex raw,
	// and auth-recipe names ("basic_auth", "magic_link_login") are not
	// regex-inert.
	_, err := collection.DeleteMany(ctx, bson.M{"user_id": bson.M{"$regex": "(?s)^" + regexp.QuoteMeta(namespace) + ":"}})
	return err
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lt": currentTime}})
	return err
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	err = cursor.All(ctx, &tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session tokens: %w", err)
	}
	return tokens, nil
}
