package graphql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ResendOTP is the method to resend OTP.
// Permissions: none
func (g *graphqlProvider) ResendOTP(ctx context.Context, params *model.ResendOTPRequest) (*model.Response, error) {
	email := strings.ToLower(strings.Trim(refs.StringValue(params.Email), " "))
	phoneNumber := strings.Trim(refs.StringValue(params.PhoneNumber), " ")
	log := g.Log.With().Str("func", "ResendOTP").Str("email", email).Str("phone_number", phoneNumber).Logger()
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, errors.New("email or phone number is required")
	}
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	var user *schemas.User
	var isEmailServiceEnabled, isSMSServiceEnabled bool
	if email != "" {
		isEmailServiceEnabled = g.Config.IsEmailServiceEnabled
		if !isEmailServiceEnabled {
			log.Debug().Msg("Email service not enabled")
			return nil, errors.New("email service not enabled")
		}
		user, err = g.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
			return nil, fmt.Errorf(`user with this email/phone not found`)
		}
	} else {
		isSMSServiceEnabled = g.Config.IsSMSServiceEnabled
		if !isSMSServiceEnabled {
			log.Debug().Msg("SMS service not enabled")
			return nil, errors.New("email service not enabled")
		}
		user, err = g.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
			return nil, fmt.Errorf(`user with this email/phone not found`)
		}
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}

	if !refs.BoolValue(user.IsMultiFactorAuthEnabled) && user.EmailVerifiedAt != nil && user.PhoneNumberVerifiedAt != nil {
		log.Debug().Msg("Multi factor authentication not enabled")
		return nil, fmt.Errorf(`multi factor authentication not enabled`)
	}

	isMFADisabled := g.Config.DisableMFA
	if isMFADisabled {
		log.Debug().Msg("Multi factor authentication is disabled for this instance")
		return nil, errors.New("multi factor authentication is disabled for this instance")
	}

	// get otp by email or phone number
	var otpData *schemas.OTP
	if email != "" {
		otpData, err = g.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
		log.Debug().Msg("Failed to get otp for given email")
	} else {
		otpData, err = g.StorageProvider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
		log.Debug().Msg("Failed to get otp for given phone number")
	}
	if err != nil {
		return nil, err
	}
	if otpData == nil {
		log.Debug().Msg("Failed to get otp for given email")
		return &model.Response{
			Message: "Failed to get for given email",
		}, errors.New("failed to get otp for given email")
	}
	// If multi factor authentication is enabled and we need to generate OTP for mail / sms based MFA
	generateOTP := func(expiresAt int64) (*schemas.OTP, error) {
		otp := utils.GenerateOTP()
		otpData, err := g.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         otp,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Msg("Failed to upsert otp")
			return nil, err
		}
		return otpData, nil
	}
	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = g.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return err
		}
		cookie.SetMfaSession(gc, mfaSession)
		return nil
	}
	expiresAt := time.Now().Add(1 * time.Minute).Unix()
	otpData, err = generateOTP(expiresAt)
	if err != nil {
		log.Debug().Msg("Failed to generate otp")
		return nil, err
	}
	if err := setOTPMFaSession(expiresAt); err != nil {
		log.Debug().Err(err).Msg("Failed to set mfa session")
		return nil, err
	}
	if email != "" {
		go func() {
			// exec it as go routine so that we can reduce the api latency
			if err := g.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(g.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Err(err).Msg("Failed to send email")
			}
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()
	} else {
		go func() {
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := g.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug().Err(err).Msg("Failed to send sms")
			}
		}()
	}
	log.Info().Msg("OTP has been sent")
	return &model.Response{
		Message: `OTP has been sent. Please check your inbox`,
	}, nil
}
