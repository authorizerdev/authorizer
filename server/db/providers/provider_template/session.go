package provider_template

import (
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddSession to save session information in database
func (p *provider) AddSession(session models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(userId string) error {
	return nil
}
