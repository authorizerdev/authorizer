package mongodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UpsertOTP to add or update otp
func (p *provider) UpsertOTP(ctx context.Context, otp *models.OTP) (*models.OTP, error) {
	if otp.ID == "" {
		otp.ID = uuid.New().String()
	}

	otp.Key = otp.ID
	if otp.CreatedAt <= 0 {
		otp.CreatedAt = time.Now().Unix()
	}
	otp.UpdatedAt = time.Now().Unix()

	otpCollection := p.db.Collection(models.Collections.OTP, options.Collection())
	_, err := otpCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": otp.ID}}, bson.M{"$set": otp}, options.MergeUpdateOptions().SetUpsert(true))
	if err != nil {
		return nil, err
	}

	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error) {
	var otp models.OTP

	otpCollection := p.db.Collection(models.Collections.OTP, options.Collection())
	err := otpCollection.FindOne(ctx, bson.M{"email": emailAddress}).Decode(&otp)
	if err != nil {
		return nil, err
	}

	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *models.OTP) error {
	otpCollection := p.db.Collection(models.Collections.OTP, options.Collection())
	_, err := otpCollection.DeleteOne(nil, bson.M{"_id": otp.ID}, options.Delete())
	if err != nil {
		return err
	}

	return nil
}
