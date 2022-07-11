package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
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

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	query := fmt.Sprintf(`FOR d IN %s FILTER d.user_id == @userId REMOVE { _key: d._key } IN %s`, models.Collections.Session, models.Collections.Session)
	bindVars := map[string]interface{}{
		"userId": userId,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}
