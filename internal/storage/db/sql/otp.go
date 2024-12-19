package sql

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *schemas.OTP) (*schemas.OTP, error) {
	if otpParam.ID == "" {
		otpParam.ID = uuid.New().String()
	}
	// check if email or phone number is present
	if otpParam.Email == "" && otpParam.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}
	uniqueField := schemas.FieldNameEmail
	if otpParam.Email == "" && otpParam.PhoneNumber != "" {
		uniqueField = schemas.FieldNamePhoneNumber
	}
	var otp *schemas.OTP
	if uniqueField == schemas.FieldNameEmail {
		otp, _ = p.GetOTPByEmail(ctx, otpParam.Email)
	} else {
		otp, _ = p.GetOTPByPhoneNumber(ctx, otpParam.PhoneNumber)
	}
	shouldCreate := false
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
		shouldCreate = true
	} else {
		otp.Otp = otpParam.Otp
		otp.ExpiresAt = otpParam.ExpiresAt
	}
	otp.UpdatedAt = time.Now().Unix()
	if shouldCreate {
		result := p.db.Create(&otp)
		if result.Error != nil {
			return nil, result.Error
		}
	} else {
		result := p.db.Save(&otp)
		if result.Error != nil {
			return nil, result.Error
		}
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	var otp schemas.OTP
	result := p.db.Where("email = ?", emailAddress).First(&otp)
	if result.Error != nil {
		return nil, result.Error
	}
	return &otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	var otp schemas.OTP
	result := p.db.Where("phone_number = ?", phoneNumber).First(&otp)
	if result.Error != nil {
		return nil, result.Error
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	result := p.db.Delete(&schemas.OTP{
		ID: otp.ID,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
