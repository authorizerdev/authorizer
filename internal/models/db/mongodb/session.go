package mongodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session *models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	session.Key = session.ID
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	sessionCollection := p.db.Collection(models.Collections.Session, options.Collection())
	_, err := sessionCollection.InsertOne(ctx, session)
	if err != nil {
		return err
	}
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	return nil
}
