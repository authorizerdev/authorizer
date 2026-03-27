package cassandradb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
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
	query := ""
	if shouldCreate {
		query = fmt.Sprintf(`INSERT INTO %s (id, email, phone_number, otp, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, KeySpace+"."+schemas.Collections.OTP)
		err := p.db.Query(query, otp.ID, otp.Email, otp.PhoneNumber, otp.Otp, otp.ExpiresAt, otp.CreatedAt, otp.UpdatedAt).Exec()
		if err != nil {
			return nil, err
		}
	} else {
		query = fmt.Sprintf(`UPDATE %s SET otp = ?, expires_at = ?, updated_at = ? WHERE id = ?`, KeySpace+"."+schemas.Collections.OTP)
		err := p.db.Query(query, otp.Otp, otp.ExpiresAt, otp.UpdatedAt, otp.ID).Exec()
		if err != nil {
			return nil, err
		}
	}

	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	var otp schemas.OTP
	query := fmt.Sprintf(`SELECT id, email, phone_number, otp, expires_at, created_at, updated_at FROM %s WHERE email = ? LIMIT 1 ALLOW FILTERING`, KeySpace+"."+schemas.Collections.OTP)
	err := p.db.Query(query, emailAddress).Consistency(gocql.One).Scan(&otp.ID, &otp.Email, &otp.PhoneNumber, &otp.Otp, &otp.ExpiresAt, &otp.CreatedAt, &otp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	var otp schemas.OTP
	query := fmt.Sprintf(`SELECT id, email, phone_number, otp, expires_at, created_at, updated_at FROM %s WHERE phone_number = ? LIMIT 1 ALLOW FILTERING`, KeySpace+"."+schemas.Collections.OTP)
	err := p.db.Query(query, phoneNumber).Consistency(gocql.One).Scan(&otp.ID, &otp.Email, &otp.PhoneNumber, &otp.Otp, &otp.ExpiresAt, &otp.CreatedAt, &otp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.OTP)
	err := p.db.Query(query, otp.ID).Exec()
	if err != nil {
		return err
	}
	return nil
}
