package surrealdb

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// UpsertOTP saves an OTP, replacing the one already held for the same email
// (or, without an email, the same phone number).
func (p *provider) UpsertOTP(ctx context.Context, otpParam *schemas.OTP) (*schemas.OTP, error) {
	if otpParam.Email == "" && otpParam.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}
	var otp *schemas.OTP
	if otpParam.Email != "" {
		otp, _ = p.GetOTPByEmail(ctx, otpParam.Email)
	} else {
		otp, _ = p.GetOTPByPhoneNumber(ctx, otpParam.PhoneNumber)
	}
	if otp == nil {
		id := uuid.NewString()
		otp = &schemas.OTP{
			ID:          id,
			Key:         id,
			Otp:         otpParam.Otp,
			Email:       otpParam.Email,
			PhoneNumber: otpParam.PhoneNumber,
			ExpiresAt:   otpParam.ExpiresAt,
			CreatedAt:   time.Now().Unix(),
		}
		otp.UpdatedAt = time.Now().Unix()
		if err := p.create(ctx, schemas.Collections.OTP, otp.ID, otp); err != nil {
			return nil, err
		}
		return otp, nil
	}
	otp.Otp = otpParam.Otp
	otp.ExpiresAt = otpParam.ExpiresAt
	otp.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.OTP, otp.ID, otp); err != nil {
		return nil, err
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	return getWhere[schemas.OTP](ctx, p, schemas.Collections.OTP, "email = $v", "otp", map[string]any{"v": emailAddress})
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	return getWhere[schemas.OTP](ctx, p, schemas.Collections.OTP, "phone_number = $v", "otp", map[string]any{"v": phoneNumber})
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	return p.deleteByID(ctx, schemas.Collections.OTP, otp.ID)
}
