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
	// otp, _ = p.GetOTPByEmail(ctx, otp.Email)
	// if otp == nil {
	// 	id := uuid.NewString()
	// 	otp = &models.OTP{
	// 		ID:        id,
	// 		Key:       id,
	// 		Otp:       otp.Otp,
	// 		Email:     otp.Email,
	// 		ExpiresAt: otp.ExpiresAt,
	// 		CreatedAt: time.Now().Unix(),
	// 	}
	// }

	// otp.UpdatedAt = time.Now().Unix()
	// unsertOpt := gocb.UpsertOptions{
	// 	Context: ctx,
	// }
	// _, err := p.db.Collection(models.Collections.OTP).Upsert(otp.ID, otp, &unsertOpt)
	// if err != nil {
	// 	return nil, err
	// }

	// return otp, nil
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
		query := fmt.Sprintf(`UPDATE auth._default.%s SET otp="%s", expires_at=%d, updated_at=%d WHERE _id="%s"`, models.Collections.OTP, otp.Otp, otp.ExpiresAt, otp.UpdatedAt, otp.ID)
		scope := p.db.Scope("_default")
		_, err := scope.Query(query, &gocb.QueryOptions{})
		if err != nil {
			return otp, err
		}
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	otp := models.OTP{}
	query := fmt.Sprintf(`SELECT _id, email, otp, expires_at, created_at, updated_at FROM auth._default.%s WHERE email = '%s' LIMIT 1`, models.Collections.OTP, emailAddress)
	q, err := p.db.Scope("_default").Query(query, &gocb.QueryOptions{
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
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
