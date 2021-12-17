package db

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Role struct {
	ID   uuid.UUID `gorm:"primaryKey;type:char(36)"`
	Role string    `gorm:"unique"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID = uuid.New()

	return
}

// SaveRoles function to save roles
func (mgr *manager) SaveRoles(roles []Role) error {
	res := mgr.sqlDB.Clauses(
		clause.OnConflict{
			DoNothing: true,
		}).Create(&roles)
	if res.Error != nil {
		log.Println(`Error saving roles`)
		return res.Error
	}

	return nil
}
