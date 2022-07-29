package resolvers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ResendOTPResolver is a resolver for resend otp mutation
func ResendOTPResolver(ctx context.Context, params model.ResendOTPRequest) (*model.Response, error) {
	log := log.WithFields(log.Fields{
		"email": params.Email,
	})
	params.Email = strings.ToLower(params.Email)
	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return nil, fmt.Errorf(`user with this email not found`)
	}

	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}

	if user.EmailVerifiedAt == nil {
		log.Debug("User email is not verified")
		return nil, fmt.Errorf(`email not verified`)
	}

	if !refs.BoolValue(user.IsMultiFactorAuthEnabled) {
		log.Debug("User multi factor authentication is not enabled")
		return nil, fmt.Errorf(`multi factor authentication not enabled`)
	}

	// get otp by email
	otpData, err := db.Provider.GetOTPByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get otp for given email: ", err)
		return nil, err
	}

	if otpData == nil {
		log.Debug("No otp found for given email: ", params.Email)
		return &model.Response{
			Message: "Failed to get for given email",
		}, errors.New("failed to get otp for given email")
	}

	otp := utils.GenerateOTP()
	otpData, err = db.Provider.UpsertOTP(ctx, &models.OTP{
		Email:     user.Email,
		Otp:       otp,
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	})
	if err != nil {
		log.Debug("Error generating new otp: ", err)
		return nil, err
	}

	go func() {
		err := email.SendOtpMail(params.Email, otp)
		if err != nil {
			log.Debug("Error sending otp email: ", otp)
		}
	}()

	return &model.Response{
		Message: `OTP has been sent. Please check your inbox`,
	}, nil
}
