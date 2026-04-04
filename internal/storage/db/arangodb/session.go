package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
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
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	query := fmt.Sprintf("FOR s IN %s FILTER s.user_id == @userId REMOVE s IN %s", schemas.Collections.Session, schemas.Collections.Session)
	bindVars := map[string]interface{}{
		"userId": userId,
	}
	_, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	return nil
}
