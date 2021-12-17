package db

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Session struct {
	ID        uuid.UUID `gorm:"primaryKey;type:char(36)"`
	UserID    uuid.UUID `gorm:"type:char(36)"`
	User      User
	UserAgent string
	IP        string
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`
}

func (r *Session) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID = uuid.New()

	return
}

// SaveSession function to save user sessiosn
func (mgr *manager) SaveSession(session Session) error {
	res := mgr.sqlDB.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&session)
	if res.Error != nil {
		log.Println(`Error saving session`, res.Error)
		return res.Error
	}

	return nil
}
