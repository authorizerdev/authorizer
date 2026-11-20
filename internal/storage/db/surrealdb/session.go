package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session *schemas.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.Key = session.ID
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	return p.create(ctx, schemas.Collections.Session, session.ID, session)
}

// DeleteSession deletes every session of a user.
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	return p.exec(ctx, "DELETE type::table($tb) WHERE user_id = $u RETURN NONE;",
		map[string]any{"tb": schemas.Collections.Session, "u": userId})
}
