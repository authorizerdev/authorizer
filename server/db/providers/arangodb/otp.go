package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
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

	otp.UpdatedAt = time.Now().Unix()
	otpCollection, _ := p.db.Collection(ctx, models.Collections.OTP)

	var meta driver.DocumentMeta
	var err error
	if shouldCreate {
		meta, err = otpCollection.CreateDocument(ctx, otp)
	} else {
		meta, err = otpCollection.UpdateDocument(ctx, otp.Key, otp)
	}

	if err != nil {
		return nil, err
	}

	otp.Key = meta.Key
	otp.ID = meta.ID.String()
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	var otp models.OTP
	query := fmt.Sprintf("FOR d in %s FILTER d.email == @email RETURN d", models.Collections.OTP)
	bindVars := map[string]interface{}{
		"email": emailAddress,
	}

	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if otp.Key == "" {
				return nil, fmt.Errorf("email template not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &otp)
		if err != nil {
			return nil, err
		}
	}

	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	otpCollection, _ := p.db.Collection(ctx, models.Collections.OTP)
	_, err := otpCollection.RemoveDocument(ctx, otp.ID)
	if err != nil {
		return err
	}

	return nil
}
