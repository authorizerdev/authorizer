package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ForgotPassword is a  for forgot password mutation
// It sends an email or sms with a verification token
// Permissions: none
func (s *service) ForgotPassword(ctx context.Context, params *model.ForgotPasswordInput) (*model.ForgotPasswordResponse, error) {
	log := s.Log.With().Str("func", "ForgotPassword").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	isBasicAuthDisabled := s.Config.DisableStrongPassword
	isEmailVerificationDisabled := s.Config.DisableEmailVerification
	isMobileBasicAuthDisabled := s.Config.DisableMobileBasicAuthentication
	isMobileVerificationDisabled := s.Config.DisablePhoneVerification
	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, fmt.Errorf(`email or phone number is required`)
	}
	log = log.With().Str("email", email).Str("phoneNumber", phoneNumber).Logger()
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if isBasicAuthDisabled && isEmailLogin && !isEmailVerificationDisabled {
		log.Debug().Msgf("Basic authentication is disabled.")
		return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileLogin && !isMobileVerificationDisabled {
		log.Debug().Msgf("Mobile basic authentication is disabled.")
		return nil, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	var user *schemas.User
	if isEmailLogin {
		user, err = s.StorageProvider.GetUserByEmail(ctx, email)
		log.Debug().Err(err).Msg("Failed to get user by email")
	} else {
		user, err = s.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		log.Debug().Err(err).Msg("Failed to get user by phone number")
	}
	if err != nil {
		return nil, fmt.Errorf(`bad user credentials`)
	}
	hostname := parsers.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to generate nonce")
		return nil, err
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}
	if isEmailLogin {
		redirectURI := ""
		// give higher preference to params redirect uri
		if strings.TrimSpace(refs.StringValue(params.RedirectURI)) != "" {
			redirectURI = refs.StringValue(params.RedirectURI)
		} else {
			redirectURI = s.Config.ResetPasswordURL
			if redirectURI == "" {
				log.Debug().Err(err).Msg("Failed to get reset password url")
				redirectURI = hostname + "/app/reset-password"
			}
		}

		verificationToken, err := s.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
			Nonce:       nonceHash,
			User:        user,
			HostName:    hostname,
		}, redirectURI, constants.VerificationTypeForgotPassword)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create verification token")
			return nil, err
		}
		_, err = s.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  constants.VerificationTypeForgotPassword,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURI,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, err
		}
		// execute it as go routine so that we can reduce the api latency
		go s.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeForgotPassword, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(s.Config),
			"verification_url": utils.GetForgotPasswordURL(verificationToken, redirectURI),
		})
		return &model.ForgotPasswordResponse{
			Message: `Please check your inbox! We have sent a password reset link.`,
		}, nil
	}
	if isMobileLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otp := utils.GenerateOTP()
		otpData, err := s.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         otp,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to upsert otp")
			return nil, err
		}
		mfaSession := uuid.NewString()
		err = s.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return nil, err
		}
		cookie.SetMfaSession(gc, mfaSession)
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(otpData.Otp)
		if err := s.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
			log.Debug().Err(err).Msg("Failed to send sms")
			// continue
		}
		return &model.ForgotPasswordResponse{
			Message:                   "Please enter the OTP sent to your phone number and change your password.",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, nil
	}
	return nil, fmt.Errorf(`email or phone number verification needs to be enabled`)
}
