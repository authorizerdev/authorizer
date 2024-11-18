package couchbase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
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
		shouldCreate = true
		otp = &schemas.OTP{
			ID:          uuid.NewString(),
			Otp:         otpParam.Otp,
			Email:       otpParam.Email,
			PhoneNumber: otpParam.PhoneNumber,
			ExpiresAt:   otpParam.ExpiresAt,
			CreatedAt:   time.Now().Unix(),
			UpdatedAt:   time.Now().Unix(),
		}
	} else {
		otp.Otp = otpParam.Otp
		otp.ExpiresAt = otpParam.ExpiresAt
	}
	otp.UpdatedAt = time.Now().Unix()
	if shouldCreate {
		insertOpt := gocb.InsertOptions{
			Context: ctx,
		}
		_, err := p.db.Collection(schemas.Collections.OTP).Insert(otp.ID, otp, &insertOpt)
		if err != nil {
			return nil, err
		}
	} else {
		query := fmt.Sprintf(`UPDATE %s.%s SET otp=$1, expires_at=$2, updated_at=$3 WHERE _id=$4`, p.scopeName, schemas.Collections.OTP)
		_, err := p.db.Query(query, &gocb.QueryOptions{
			PositionalParameters: []interface{}{otp.Otp, otp.ExpiresAt, otp.UpdatedAt, otp.ID},
		})
		if err != nil {
			return nil, err
		}
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	otp := schemas.OTP{}
	query := fmt.Sprintf(`SELECT _id, email, phone_number, otp, expires_at, created_at, updated_at FROM %s.%s WHERE email = $1 LIMIT 1`, p.scopeName, schemas.Collections.OTP)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{emailAddress},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&otp)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	otp := schemas.OTP{}
	query := fmt.Sprintf(`SELECT _id, email, phone_number, otp, expires_at, created_at, updated_at FROM %s.%s WHERE phone_number = $1 LIMIT 1`, p.scopeName, schemas.Collections.OTP)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{phoneNumber},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&otp)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.OTP).Remove(otp.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
