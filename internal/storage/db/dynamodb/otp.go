package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *schemas.OTP) (*schemas.OTP, error) {
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
	var err error
	if shouldCreate {
		err = p.putItem(ctx, schemas.Collections.OTP, otp)
	} else {
		err = p.updateByHashKey(ctx, schemas.Collections.OTP, "id", otp.ID, otp)
	}
	if err != nil {
		return nil, err
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.OTP, "email", "email", emailAddress, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no docuemnt found")
	}
	var otp schemas.OTP
	if err := unmarshalItem(items[0], &otp); err != nil {
		return nil, err
	}
	return &otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	f := expression.Name("phone_number").Equal(expression.Value(phoneNumber))
	items, err := p.scanAllRaw(ctx, schemas.Collections.OTP, nil, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no docuemnt found")
	}
	var otp schemas.OTP
	if err := unmarshalItem(items[0], &otp); err != nil {
		return nil, err
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	if otp == nil || otp.ID == "" {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.OTP, "id", otp.ID)
}
