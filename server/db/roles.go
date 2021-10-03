package db

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;"`
	Role string    `gorm:"unique"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID = uuid.New()

	return
}

// SaveRoles function to save roles
func (mgr *manager) SaveRoles(roles []Role) error {
	res := mgr.db.Clauses(
		clause.OnConflict{
			OnConstraint: "authorizer_roles_role_key",
			DoNothing:    true,
		}).Create(&roles)
	if res.Error != nil {
		log.Println(`Error saving roles`)
		return res.Error
	}

	return nil
}
