package db

import "log"

type Role struct {
	ID   uint `gorm:"primaryKey"`
	Role string
}

// SaveRoles function to save roles
func (mgr *manager) SaveRoles(roles []Role) error {
	res := mgr.db.Create(&roles)
	if res.Error != nil {
		log.Println(`Error saving roles`)
		return res.Error
	}

	return nil
}
