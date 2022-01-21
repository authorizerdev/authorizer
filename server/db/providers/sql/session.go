package sql

import (
	"log"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// AddSession to save session information in database
func (p *provider) AddSession(session models.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	session.Key = session.ID
	res := p.db.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&session)
	if res.Error != nil {
		log.Println(`error saving session`, res.Error)
		return res.Error
	}
	return nil
}

// DeleteSession to delete session information from database
func (p *provider) DeleteSession(userId string) error {
	result := p.db.Where("user_id = ?", userId).Delete(&models.Session{})

	if result.Error != nil {
		log.Println(`error deleting session:`, result.Error)
		return result.Error
	}
	return nil
}
