package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *models.OTP) (*models.OTP, error) {
	otp, _ := p.GetOTPByEmail(ctx, otpParam.Email)
	shouldCreate := false
	if otp == nil {
		shouldCreate = true
		otp.ID = uuid.New().String()
		otp.Key = otp.ID
		otp.CreatedAt = time.Now().Unix()
	} else {
		otp = otpParam
	}

	otp.UpdatedAt = time.Now().Unix()
	otpCollection, _ := p.db.Collection(ctx, models.Collections.OTP)

	if shouldCreate {
		_, err := otpCollection.CreateDocument(ctx, otp)
		if err != nil {
			return nil, err
		}
	} else {
		meta, err := otpCollection.UpdateDocument(ctx, otp.Key, otp)
		if err != nil {
			return nil, err
		}

		otp.Key = meta.Key
		otp.ID = meta.ID.String()
	}

	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	var otp *models.OTP
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
		_, err := cursor.ReadDocument(ctx, otp)
		if err != nil {
			return nil, err
		}
	}

	return otp, nil
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
