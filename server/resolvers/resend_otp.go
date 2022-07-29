package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ResendOTPResolver is a resolver for resend otp mutation
func ResendOTPResolver(ctx context.Context, params model.ResendOTPRequest) (*model.Response, error) {
	var res *model.Response

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})
	params.Email = strings.ToLower(params.Email)
	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return res, fmt.Errorf(`user with this email not found`)
	}

	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
	}

	if user.EmailVerifiedAt == nil {
		log.Debug("User email is not verified")
		return res, fmt.Errorf(`email not verified`)
	}

	if !refs.BoolValue(user.IsMultiFactorAuthEnabled) {
		log.Debug("User multi factor authentication is not enabled")
		return res, fmt.Errorf(`multi factor authentication not enabled`)
	}

	//TODO - send email based on email config
	db.Provider.UpsertOTP(ctx, &models.OTP{
		Email:     user.Email,
		Otp:       utils.GenerateOTP(),
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	})

	res = &model.Response{
		Message: `OTP has been sent. Please check your inbox`,
	}

	return res, nil
}
