package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/gocql/gocql"
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
	query := ""

	if shouldCreate {
		query = fmt.Sprintf(`INSERT INTO %s (id, email, otp, expires_at, created_at, updated_at) VALUES ('%s', '%s', '%s', %d, %d, %d)`, KeySpace+"."+models.Collections.OTP, otp.ID, otp.Email, otp.Otp, otp.ExpiresAt, otp.CreatedAt, otp.UpdatedAt)
	} else {
		query = fmt.Sprintf(`UPDATE %s SET otp = '%s', expires_at = %d, updated_at = %d WHERE email = '%s'`, KeySpace+"."+models.Collections.OTP, otp.Otp, otp.ExpiresAt, otp.UpdatedAt, otp.Email)
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
	query := fmt.Sprintf(`SELECT id, email, otp, expires_at, created_at, updated_at FROM %s WHERE email = '%s' LIMIT 1 ALLOW FILTERING`, KeySpace+"."+models.Collections.OTP, emailAddress)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&otp.ID, &otp.Email, &otp.Otp, &otp.ExpiresAt, &otp.CreatedAt, &otp.UpdatedAt)
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
