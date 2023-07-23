package resolvers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	emailHelper "github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/smsproviders"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ResendOTPResolver is a resolver for resend otp mutation
func ResendOTPResolver(ctx context.Context, params model.ResendOTPRequest) (*model.Response, error) {
	email := strings.ToLower(strings.Trim(refs.StringValue(params.Email), " "))
	phoneNumber := strings.Trim(refs.StringValue(params.PhoneNumber), " ")
	log := log.WithFields(log.Fields{
		"email":        email,
		"phone_number": phoneNumber,
	})
	if email == "" && phoneNumber == "" {
		log.Debug("Email or phone number is required")
		return nil, errors.New("email or phone number is required")
	}
	var user models.User
	var err error
	if email != "" {
		isEmailServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
		if err != nil || !isEmailServiceEnabled {
			log.Debug("Email service not enabled: ", err)
			return nil, errors.New("email service not enabled")
		}
		user, err = db.Provider.GetUserByEmail(ctx, email)
	} else {
		isSMSServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
		if err != nil || !isSMSServiceEnabled {
			log.Debug("Email service not enabled: ", err)
			return nil, errors.New("email service not enabled")
		}
		// TODO fix after refs fixes
		var u *models.User
		u, err = db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
		user = *u
	}
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return nil, fmt.Errorf(`user with this email/phone not found`)
	}

	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}

	if email != "" && user.EmailVerifiedAt == nil {
		log.Debug("User email is not verified")
		return nil, fmt.Errorf(`email not verified`)
	}

	if phoneNumber != "" && user.PhoneNumberVerifiedAt == nil {
		log.Debug("User phone number is not verified")
		return nil, fmt.Errorf(`phone number not verified`)
	}

	if !refs.BoolValue(user.IsMultiFactorAuthEnabled) {
		log.Debug("User multi factor authentication is not enabled")
		return nil, fmt.Errorf(`multi factor authentication not enabled`)
	}

	isMFADisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil || isMFADisabled {
		log.Debug("MFA service not enabled: ", err)
		return nil, errors.New("multi factor authentication is disabled for this instance")
	}

	// get otp by email or phone number
	var otpData *models.OTP
	if email != "" {
		otpData, err = db.Provider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
	} else {
		otpData, err = db.Provider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
	}
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
	if _, err := db.Provider.UpsertOTP(ctx, &models.OTP{
		Email:     user.Email,
		Otp:       otp,
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	}); err != nil {
		log.Debug("Error upserting otp: ", err)
		return nil, err
	}

	if email != "" {
		// exec it as go routine so that we can reduce the api latency
		go emailHelper.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
			"user":         user.ToMap(),
			"organization": utils.GetOrganization(),
			"otp":          otp,
		})
	} else {
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(otp)
		// exec it as go routine so that we can reduce the api latency
		go smsproviders.SendSMS(phoneNumber, smsBody.String())
	}
	log.Info("OTP has been resent")
	return &model.Response{
		Message: `OTP has been sent. Please check your inbox`,
	}, nil
}
