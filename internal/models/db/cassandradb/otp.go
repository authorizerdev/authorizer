package cassandradb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *models.OTP) (*models.OTP, error) {
	// check if email or phone number is present
	if otpParam.Email == "" && otpParam.PhoneNumber == "" {
		return nil, errors.New("email or phone_number is required")
	}
	uniqueField := models.FieldNameEmail
	if otpParam.Email == "" && otpParam.PhoneNumber != "" {
		uniqueField = models.FieldNamePhoneNumber
	}
	var otp *models.OTP
	if uniqueField == models.FieldNameEmail {
		otp, _ = p.GetOTPByEmail(ctx, otpParam.Email)
	} else {
		otp, _ = p.GetOTPByPhoneNumber(ctx, otpParam.PhoneNumber)
	}
	shouldCreate := false
	if otp == nil {
		shouldCreate = true
		otp = &models.OTP{
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
		query = fmt.Sprintf(`INSERT INTO %s (id, email, phone_number, otp, expires_at, created_at, updated_at) VALUES ('%s', '%s', '%s', '%s', %d, %d, %d)`, KeySpace+"."+models.Collections.OTP, otp.ID, otp.Email, otp.PhoneNumber, otp.Otp, otp.ExpiresAt, otp.CreatedAt, otp.UpdatedAt)
	} else {
		query = fmt.Sprintf(`UPDATE %s SET otp = '%s', expires_at = %d, updated_at = %d WHERE id = '%s'`, KeySpace+"."+models.Collections.OTP, otp.Otp, otp.ExpiresAt, otp.UpdatedAt, otp.ID)
	}

	err := p.db.Query(query).Exec()
	if err != nil {
		return nil, err
	}

	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	var otp models.OTP
	query := fmt.Sprintf(`SELECT id, email, phone_number, otp, expires_at, created_at, updated_at FROM %s WHERE email = '%s' LIMIT 1 ALLOW FILTERING`, KeySpace+"."+models.Collections.OTP, emailAddress)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&otp.ID, &otp.Email, &otp.PhoneNumber, &otp.Otp, &otp.ExpiresAt, &otp.CreatedAt, &otp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*models.OTP, error) {
	var otp models.OTP
	query := fmt.Sprintf(`SELECT id, email, phone_number, otp, expires_at, created_at, updated_at FROM %s WHERE phone_number = '%s' LIMIT 1 ALLOW FILTERING`, KeySpace+"."+models.Collections.OTP, phoneNumber)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&otp.ID, &otp.Email, &otp.PhoneNumber, &otp.Otp, &otp.ExpiresAt, &otp.CreatedAt, &otp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+models.Collections.OTP, otp.ID)
	err := p.db.Query(query).Exec()
	if err != nil {
		return err
	}
	return nil
}
