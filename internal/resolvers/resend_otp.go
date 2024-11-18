package resolvers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/data_store/db"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	mailService "github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/smsproviders"
	"github.com/authorizerdev/authorizer/internal/utils"
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
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}
	var user *models.User
	var isEmailServiceEnabled, isSMSServiceEnabled bool
	if email != "" {
		isEmailServiceEnabled, err = memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
		if err != nil || !isEmailServiceEnabled {
			log.Debug("Email service not enabled: ", err)
			return nil, errors.New("email service not enabled")
		}
		user, err = db.Provider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug("Failed to get user by email: ", err)
			return nil, fmt.Errorf(`user with this email/phone not found`)
		}
	} else {
		isSMSServiceEnabled, err = memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
		if err != nil || !isSMSServiceEnabled {
			log.Debug("Email service not enabled: ", err)
			return nil, errors.New("email service not enabled")
		}
		user, err = db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug("Failed to get user by phone: ", err)
			return nil, fmt.Errorf(`user with this email/phone not found`)
		}
	}
	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}

	if !refs.BoolValue(user.IsMultiFactorAuthEnabled) && user.EmailVerifiedAt != nil && user.PhoneNumberVerifiedAt != nil {
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
	// If multi factor authentication is enabled and we need to generate OTP for mail / sms based MFA
	generateOTP := func(expiresAt int64) (*models.OTP, error) {
		otp := utils.GenerateOTP()
		otpData, err := db.Provider.UpsertOTP(ctx, &models.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         otp,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug("Failed to add otp: ", err)
			return nil, err
		}
		return otpData, nil
	}
	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return err
		}
		cookie.SetMfaSession(gc, mfaSession)
		return nil
	}
	expiresAt := time.Now().Add(1 * time.Minute).Unix()
	otpData, err = generateOTP(expiresAt)
	if err != nil {
		log.Debug("Failed to generate otp: ", err)
		return nil, err
	}
	if err := setOTPMFaSession(expiresAt); err != nil {
		log.Debug("Failed to set mfa session: ", err)
		return nil, err
	}
	if email != "" {
		go func() {
			// exec it as go routine so that we can reduce the api latency
			if err := mailService.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug("Failed to send otp email: ", err)
			}
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()
	} else {
		go func() {
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := smsproviders.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug("Failed to send sms: ", err)
			}
		}()
	}
	log.Info("OTP has been resent")
	return &model.Response{
		Message: `OTP has been sent. Please check your inbox`,
	}, nil
}
