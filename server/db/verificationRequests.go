package db

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VerificationRequest struct {
	ID         uuid.UUID `gorm:"primaryKey;type:char(36)"`
	Token      string    `gorm:"type:text"`
	Identifier string
	ExpiresAt  int64
	CreatedAt  int64  `gorm:"autoCreateTime"`
	UpdatedAt  int64  `gorm:"autoUpdateTime"`
	Email      string `gorm:"unique"`
}

func (v *VerificationRequest) BeforeCreate(tx *gorm.DB) (err error) {
	v.ID = uuid.New()

	return
}

// AddVerification function to add verification record
func (mgr *manager) AddVerification(verification VerificationRequest) (VerificationRequest, error) {
	result := mgr.sqlDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"token", "identifier", "expires_at"}),
	}).Create(&verification)

	if result.Error != nil {
		log.Println(`Error saving verification record`, result.Error)
		return verification, result.Error
	}
	return verification, nil
}

func (mgr *manager) GetVerificationByToken(token string) (VerificationRequest, error) {
	var verification VerificationRequest
	result := mgr.sqlDB.Where("token = ?", token).First(&verification)

	if result.Error != nil {
		log.Println(`Error getting verification token:`, result.Error)
		return verification, result.Error
	}

	return verification, nil
}

func (mgr *manager) GetVerificationByEmail(email string) (VerificationRequest, error) {
	var verification VerificationRequest
	result := mgr.sqlDB.Where("email = ?", email).First(&verification)

	if result.Error != nil {
		log.Println(`Error getting verification token:`, result.Error)
		return verification, result.Error
	}

	return verification, nil
}

func (mgr *manager) DeleteToken(email string) error {
	var verification VerificationRequest
	result := mgr.sqlDB.Where("email = ?", email).Delete(&verification)

	if result.Error != nil {
		log.Println(`Error deleting token:`, result.Error)
		return result.Error
	}

	return nil
}

// GetUsers function to get all users
func (mgr *manager) GetVerificationRequests() ([]VerificationRequest, error) {
	var verificationRequests []VerificationRequest
	result := mgr.sqlDB.Find(&verificationRequests)
	if result.Error != nil {
		log.Println(result.Error)
		return verificationRequests, result.Error
	}
	return verificationRequests, nil
}
