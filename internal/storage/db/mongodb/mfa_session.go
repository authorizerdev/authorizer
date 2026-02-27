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
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	_, err := collection.InsertOne(ctx, session)
	return err
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	var session schemas.MFASession
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	err := collection.FindOne(ctx, bson.M{"user_id": userId, "key_name": key}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"user_id": userId, "key_name": key})
	return err
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{"user_id": userId})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MFA sessions: %w", err)
	}
	return sessions, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lt": currentTime}})
	return err
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	collection := p.db.Collection(schemas.Collections.MFASession, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MFA sessions: %w", err)
	}
	return sessions, nil
}
