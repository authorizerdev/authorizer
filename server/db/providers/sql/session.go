package sql

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// AddSession to save session information in database
func (p *provider) AddSession(ctx context.Context, session models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	session.Key = session.ID
	session.CreatedAt = time.Now().Unix()
	session.UpdatedAt = time.Now().Unix()
	res := p.db.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&session)
	if res.Error != nil {
		return res.Error
	}
	return nil
}
