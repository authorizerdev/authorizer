package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddLoginAttempt records a login attempt in the database
func (p *provider) AddLoginAttempt(ctx context.Context, loginAttempt *schemas.LoginAttempt) error {
	if loginAttempt.ID == "" {
		loginAttempt.ID = uuid.New().String()
	}
	loginAttempt.Key = loginAttempt.ID
	loginAttempt.CreatedAt = time.Now().Unix()
	if loginAttempt.AttemptedAt == 0 {
		loginAttempt.AttemptedAt = loginAttempt.CreatedAt
	}
	loginAttemptCollection := p.db.Collection(schemas.Collections.LoginAttempt, options.Collection())
	_, err := loginAttemptCollection.InsertOne(ctx, loginAttempt)
	return err
}

// CountFailedLoginAttemptsSince counts failed login attempts for an email since the given Unix timestamp
func (p *provider) CountFailedLoginAttemptsSince(ctx context.Context, email string, since int64) (int64, error) {
	loginAttemptCollection := p.db.Collection(schemas.Collections.LoginAttempt, options.Collection())
	query := bson.M{
		"email":       email,
		"successful":  false,
		"attempted_at": bson.M{"$gte": since},
	}
	count, err := loginAttemptCollection.CountDocuments(ctx, query, options.Count())
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteLoginAttemptsBefore removes all login attempts older than the given Unix timestamp
func (p *provider) DeleteLoginAttemptsBefore(ctx context.Context, before int64) error {
	loginAttemptCollection := p.db.Collection(schemas.Collections.LoginAttempt, options.Collection())
	query := bson.M{
		"attempted_at": bson.M{"$lt": before},
	}
	_, err := loginAttemptCollection.DeleteMany(ctx, query)
	return err
}
