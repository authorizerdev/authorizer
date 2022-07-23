package provider_template

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddOTP to add otp
func (p *provider) AddOTP(ctx context.Context, otp *models.OTP) (*models.OTP, error) {
	if otp.ID == "" {
		otp.ID = uuid.New().String()
	}

	otp.Key = otp.ID
	otp.CreatedAt = time.Now().Unix()
	otp.UpdatedAt = time.Now().Unix()

	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	return nil, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	return nil
}
