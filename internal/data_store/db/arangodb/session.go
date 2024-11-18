package arangodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session *schemas.Session) error {
	if session.ID == "" {
		session.Key = session.ID
	}
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	sessionCollection, _ := p.db.Collection(ctx, schemas.Collections.Session)
	_, err := sessionCollection.CreateDocument(ctx, session)
	if err != nil {
		return err
	}
	return nil
}

// DeleteSession to delete session information from database
// TODO: Implement this function
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	return nil
}
