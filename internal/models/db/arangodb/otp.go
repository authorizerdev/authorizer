package arangodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/models/schemas"
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
	otp.UpdatedAt = time.Now().Unix()
	otpCollection, _ := p.db.Collection(ctx, schemas.Collections.OTP)
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
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	var otp *schemas.OTP
	query := fmt.Sprintf("FOR d in %s FILTER d.email == @email RETURN d", schemas.Collections.OTP)
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
			if otp == nil {
				return nil, fmt.Errorf("otp with given email not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &otp)
		if err != nil {
			return nil, err
		}
	}
	return otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	var otp *schemas.OTP
	query := fmt.Sprintf("FOR d in %s FILTER d.phone_number == @phone_number RETURN d", schemas.Collections.OTP)
	bindVars := map[string]interface{}{
		"phone_number": phoneNumber,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if otp == nil {
				return nil, fmt.Errorf("otp with given phone_number not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &otp)
		if err != nil {
			return nil, err
		}
	}
	return otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	otpCollection, _ := p.db.Collection(ctx, schemas.Collections.OTP)
	_, err := otpCollection.RemoveDocument(ctx, otp.ID)
	if err != nil {
		return err
	}
	return nil
}
