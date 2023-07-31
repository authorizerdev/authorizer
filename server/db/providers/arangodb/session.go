package arangodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session *models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
		session.Key = session.ID
	}

	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	sessionCollection, _ := p.db.Collection(ctx, models.Collections.Session)
	_, err := sessionCollection.CreateDocument(ctx, session)
	if err != nil {
		return err
	}
	return nil
}
