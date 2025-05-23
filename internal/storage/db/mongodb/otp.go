package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	otpCollection := p.db.Collection(schemas.Collections.OTP, options.Collection())

	var err error
	if shouldCreate {
		_, err = otpCollection.InsertOne(ctx, otp)
	} else {
		_, err = otpCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": otp.ID}}, bson.M{"$set": otp}, options.MergeUpdateOptions())
	}
	if err != nil {
		return nil, err
	}
	return otp, nil
}

// GetOTPByEmail to get otp for a given email address
func (p *provider) GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error) {
	var otp schemas.OTP
	otpCollection := p.db.Collection(schemas.Collections.OTP, options.Collection())
	err := otpCollection.FindOne(ctx, bson.M{"email": emailAddress}).Decode(&otp)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// GetOTPByPhoneNumber to get otp for a given phone number
func (p *provider) GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error) {
	var otp schemas.OTP
	otpCollection := p.db.Collection(schemas.Collections.OTP, options.Collection())
	err := otpCollection.FindOne(ctx, bson.M{"phone_number": phoneNumber}).Decode(&otp)
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// DeleteOTP to delete otp
func (p *provider) DeleteOTP(ctx context.Context, otp *schemas.OTP) error {
	otpCollection := p.db.Collection(schemas.Collections.OTP, options.Collection())
	_, err := otpCollection.DeleteOne(ctx, bson.M{"_id": otp.ID}, options.Delete())
	if err != nil {
		return err
	}

	return nil
}
