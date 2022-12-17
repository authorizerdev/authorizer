package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otpParam *models.OTP) (*models.OTP, error) {
	otp, _ := p.GetOTPByEmail(ctx, otpParam.Email)

	shouldCreate := false
	if otp == nil {
		shouldCreate = true
		otp = &models.OTP{
			ID:        uuid.NewString(),
			Otp:       otpParam.Otp,
			Email:     otpParam.Email,
			ExpiresAt: otpParam.ExpiresAt,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
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
		_, err := p.db.Collection(models.Collections.OTP).Insert(otp.ID, otp, &insertOpt)
		if err != nil {
			return otp, err
		}
	} else {
		query := fmt.Sprintf(`UPDATE auth._default.%s SET otp=$1, expires_at=$2, updated_at=$3 WHERE _id=$4`, models.Collections.OTP)
		_, err := p.db.Scope("_default").Query(query, &gocb.QueryOptions{
			PositionalParameters: []interface{}{otp.Otp, otp.ExpiresAt, otp.UpdatedAt, otp.ID},
		})
		if err != nil {
			return otp, err
		}
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	otp := models.OTP{}
	query := fmt.Sprintf(`SELECT _id, email, otp, expires_at, created_at, updated_at FROM auth._default.%s WHERE email = $1 LIMIT 1`, models.Collections.OTP)
	q, err := p.db.Scope("_default").Query(query, &gocb.QueryOptions{
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

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.OTP).Remove(otp.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}
