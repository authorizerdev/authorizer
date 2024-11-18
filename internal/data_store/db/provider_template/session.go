package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session *schemas.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	return nil
}
