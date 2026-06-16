package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ResendOTP re-issues a one-time passcode for an email/SMS MFA or pending
// verification challenge. Transport-agnostic port of graphqlProvider.ResendOTP.
//
// Permissions: none.
func (p *provider) ResendOTP(ctx context.Context, meta RequestMetadata, params *model.ResendOTPRequest) (*model.Response, *ResponseSideEffects, error) {
	email := strings.ToLower(strings.Trim(refs.StringValue(params.Email), " "))
	phoneNumber := strings.Trim(refs.StringValue(params.PhoneNumber), " ")
	log := p.Log.With().Str("func", "ResendOTP").Str("email", email).Str("phone_number", phoneNumber).Logger()
	side := &ResponseSideEffects{}
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument("email or phone number is required")
	}
	var user *schemas.User
	var err error
	var isEmailServiceEnabled, isSMSServiceEnabled bool
	if email != "" {
		isEmailServiceEnabled = p.Config.IsEmailServiceEnabled
		if !isEmailServiceEnabled {
			log.Debug().Msg("Email service not enabled")
			return nil, nil, FailedPrecondition("email service not enabled")
		}
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
			return &model.Response{
				Message: "If an account exists, an OTP has been sent",
			}, nil, nil
		}
	} else {
		isSMSServiceEnabled = p.Config.IsSMSServiceEnabled
		if !isSMSServiceEnabled {
			log.Debug().Msg("SMS service not enabled")
			return nil, nil, FailedPrecondition("SMS service not enabled")
		}
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
			return &model.Response{
				Message: "If an account exists, an OTP has been sent",
			}, nil, nil
		}
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return &model.Response{
			Message: "If an account exists, an OTP has been sent",
		}, nil, nil
	}

	// Block OTP resend when MFA is disabled and both email & phone are
	// already verified — there is no pending verification that needs an OTP.
	// When MFA IS enabled, or when either email/phone is still unverified,
	// OTP resend is allowed (for MFA challenges or pending verification).
	if !refs.BoolValue(user.IsMultiFactorAuthEnabled) && user.EmailVerifiedAt != nil && user.PhoneNumberVerifiedAt != nil {
		log.Debug().Msg("Multi factor authentication not enabled")
		return nil, nil, FailedPrecondition("multi factor authentication not enabled")
	}

	isMFAEnabled := p.Config.EnableMFA
	if !isMFAEnabled {
		log.Debug().Msg("Multi factor authentication is disabled for this instance")
		return nil, nil, FailedPrecondition("multi factor authentication is disabled for this instance")
	}

	// get otp by email or phone number
	var otpData *schemas.OTP
	if email != "" {
		otpData, err = p.StorageProvider.GetOTPByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get otp for given email")
		}
	} else {
		otpData, err = p.StorageProvider.GetOTPByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get otp for given phone number")
		}
	}
	if err != nil {
		return nil, nil, err
	}
	if otpData == nil {
		log.Debug().Msg("Failed to get otp for given email")
		return &model.Response{
			Message: "Failed to get for given email",
		}, nil, errors.New("failed to get otp for given email")
	}
	// If multi factor authentication is enabled and we need to generate OTP for mail / sms based MFA
	generateOTP := func(expiresAt int64) (*schemas.OTP, error) {
		otp, err := utils.GenerateOTP()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate OTP")
			return nil, err
		}
		// Store HMAC digest; the plaintext is restored on the returned
		// struct so the caller's email/SMS body can read otpData.Otp.
		otpData, err := p.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         crypto.HashOTP(otp, p.Config.JWTSecret),
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Msg("Failed to upsert otp")
			return nil, err
		}
		otpData.Otp = otp
		return otpData, nil
	}
	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = p.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return err
		}
		for _, c := range cookie.BuildMfaSessionCookies(meta.HostURL, mfaSession, p.Config.AppCookieSecure) {
			side.AddCookie(c)
		}
		return nil
	}
	expiresAt := time.Now().Add(1 * time.Minute).Unix()
	otpData, err = generateOTP(expiresAt)
	if err != nil {
		log.Debug().Msg("Failed to generate otp")
		return nil, nil, err
	}
	if err := setOTPMFaSession(expiresAt); err != nil {
		log.Debug().Err(err).Msg("Failed to set mfa session")
		return nil, nil, err
	}
	if email != "" {
		go func() {
			// exec it as go routine so that we can reduce the api latency
			if err := p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]any{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(p.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Err(err).Msg("Failed to send email")
			}
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()
	} else {
		go func() {
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := p.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug().Err(err).Msg("Failed to send sms")
			}
		}()
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditOTPResentEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	log.Info().Msg("OTP has been sent")
	return &model.Response{
		Message: `OTP has been sent. Please check your inbox`,
	}, side, nil
}
