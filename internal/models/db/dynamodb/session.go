package dynamodb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/models/schemas"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session *schemas.Session) error {
	collection := p.db.Table(schemas.Collections.Session)
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	err := collection.Put(session).RunWithContext(ctx)
	return err
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	return nil
}
