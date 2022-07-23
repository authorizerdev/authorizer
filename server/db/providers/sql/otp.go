package sql

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otp *models.OTP) (*models.OTP, error) {
	if otp.ID == "" {
		otp.ID = uuid.New().String()
	}

	otp.Key = otp.ID
	otp.CreatedAt = time.Now().Unix()
	otp.UpdatedAt = time.Now().Unix()

	res := p.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"otp", "expires_at", "updated_at"}),
	}).Create(&otp)
	if res.Error != nil {
		return nil, res.Error
	}

	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	var otp models.OTP

	result := p.db.Where("email = ?", emailAddress).First(&otp)
	if result.Error != nil {
		return nil, result.Error
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	result := p.db.Delete(&models.OTP{
		ID: otp.ID,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
