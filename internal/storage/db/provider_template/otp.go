package provider_template

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otp *schemas.OTP) (*schemas.OTP, error) {
	return nil, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	return nil, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	return nil, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	return nil
}
