package db

import (
	"log"

	"gorm.io/gorm/clause"
)

type Verification struct {
	ID         uint `gorm:"primaryKey"`
	Token      string
	Identifier string
	ExpiresAt  int64
	CreatedAt  int64  `gorm:"autoCreateTime"`
	UpdatedAt  int64  `gorm:"autoUpdateTime"`
	Email      string `gorm:"unique"`
}

// AddVerification function to add verification record
func (mgr *manager) AddVerification(verification Verification) (Verification, error) {
	result := mgr.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"token", "identifier", "expires_at"}),
	}).Create(&verification)
	if result.Error != nil {
		log.Println(`Error saving verification record`, result.Error)
		return verification, result.Error
	}
	return verification, nil
}
