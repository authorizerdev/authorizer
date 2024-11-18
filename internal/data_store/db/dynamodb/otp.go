package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *schemas.OTP) (*schemas.OTP, error) {
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
	collection := p.db.Table(schemas.Collections.OTP)
	otp.UpdatedAt = time.Now().Unix()
	var err error
	if shouldCreate {
		err = collection.Put(otp).RunWithContext(ctx)
	} else {
		err = UpdateByHashKey(collection, "id", otp.ID, otp)
	}
	if err != nil {
		return nil, err
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	var otps []schemas.OTP
	var otp schemas.OTP
	collection := p.db.Table(schemas.Collections.OTP)
	err := collection.Scan().Index("email").Filter("'email' = ?", emailAddress).Limit(1).AllWithContext(ctx, &otps)
	if err != nil {
		return nil, err
	}
	if len(otps) > 0 {
		otp = otps[0]
		return &otp, nil
	}
	return nil, errors.New("no docuemnt found")
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	var otps []schemas.OTP
	var otp schemas.OTP
	collection := p.db.Table(schemas.Collections.OTP)
	err := collection.Scan().Filter("'phone_number' = ?", phoneNumber).Limit(1).AllWithContext(ctx, &otps)
	if err != nil {
		return nil, err
	}
	if len(otps) > 0 {
		otp = otps[0]
		return &otp, nil
	}
	return nil, errors.New("no docuemnt found")
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	collection := p.db.Table(schemas.Collections.OTP)
	if otp.ID != "" {
		err := collection.Delete("id", otp.ID).RunWithContext(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
