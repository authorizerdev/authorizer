package couchbase

import (
	"context"

	"github.com/authorizerdev/authorizer/server/db/models"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otp *models.OTP) (*models.OTP, error) {
	return nil, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	return nil, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	return nil
}
