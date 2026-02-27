package mongodb

import (
	"context"
	"fmt"
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
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"user_id": userId, "key_name": key})
	return err
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"user_id": bson.M{"$regex": userId}})
	return err
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	collection := p.db.Collection(schemas.Collections.SessionToken, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"user_id": bson.M{"$regex": "^" + namespace + ":"}})
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
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session tokens: %w", err)
	}
	return tokens, nil
}
