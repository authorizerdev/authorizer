package db

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type Session struct {
	Key       string `json:"_key,omitempty"` // for arangodb
	ObjectID  string `json:"_id,omitempty"`  // for arangodb & mongodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"id"`
	UserID    string `gorm:"type:char(36)" json:"user_id"`
	User      User   `json:"-"`
	UserAgent string `json:"user_agent"`
	IP        string `json:"ip"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"updated_at"`
}

// AddSession function to save user sessiosn
func (mgr *manager) AddSession(session Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	if session.CreatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}

	if session.UpdatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}

	if IsSQL {
		// copy id as value for fields required for mongodb & arangodb
		session.Key = session.ID
		session.ObjectID = session.ID
		res := mgr.sqlDB.Clauses(
			clause.OnConflict{
				DoNothing: true,
			}).Create(&session)
		if res.Error != nil {
			log.Println(`Error saving session`, res.Error)
			return res.Error
		}
	}

	if IsArangoDB {

		session.CreatedAt = time.Now().Unix()
		session.UpdatedAt = time.Now().Unix()
		sessionCollection, _ := mgr.arangodb.Collection(nil, Collections.Session)
		_, err := sessionCollection.CreateDocument(nil, session)
		if err != nil {
			return err
		}
	}

	return nil
}
