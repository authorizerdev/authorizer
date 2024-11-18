package cassandradb

import (
	"context"
	"fmt"
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
	insertSessionQuery := fmt.Sprintf("INSERT INTO %s (id, user_id, user_agent, ip, created_at, updated_at) VALUES ('%s', '%s', '%s', '%s', %d, %d)", KeySpace+"."+schemas.Collections.Session, session.ID, session.UserID, session.UserAgent, session.IP, session.CreatedAt, session.UpdatedAt)
	err := p.db.Query(insertSessionQuery).Exec()
	if err != nil {
		return err
	}
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(ctx context.Context, userId string) error {
	return nil
}
