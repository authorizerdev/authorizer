package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *models.OTP) (*models.OTP, error) {
	otp, _ := p.GetOTPByEmail(ctx, otpParam.Email)
	shouldCreate := false
	if otp == nil {
		id := uuid.NewString()
		otp = &models.OTP{
			ID:        id,
			Key:       id,
			Otp:       otpParam.Otp,
			Email:     otpParam.Email,
			ExpiresAt: otpParam.ExpiresAt,
			CreatedAt: time.Now().Unix(),
		}
		shouldCreate = true
	} else {
		otp.Otp = otpParam.Otp
		otp.ExpiresAt = otpParam.ExpiresAt
	}

	collection := p.db.Table(models.Collections.OTP)
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
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	var otps []models.OTP
	var otp models.OTP

	collection := p.db.Table(models.Collections.OTP)

	err := collection.Scan().Filter("'email' = ?", emailAddress).Limit(1).AllWithContext(ctx, &otps)

	if err != nil {
		return nil, err
	}
	if len(otps) > 0 {
		otp = otps[0]
		return &otp, nil
	} else {
		return nil, errors.New("no docuemnt found")
	}
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	collection := p.db.Table(models.Collections.OTP)

	if otp.ID != "" {
		err := collection.Delete("id", otp.ID).RunWithContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
