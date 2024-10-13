package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	mailService "github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/smsproviders"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ForgotPasswordResolver is a resolver for forgot password mutation
func ForgotPasswordResolver(ctx context.Context, params model.ForgotPasswordInput) (*model.ForgotPasswordResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
	if err != nil {
		log.Debug("Error getting email verification disabled: ", err)
		isEmailVerificationDisabled = true
	}

	isMobileBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isMobileBasicAuthDisabled = true
	}
	isMobileVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if err != nil {
		log.Debug("Error getting mobile verification disabled: ", err)
		isMobileVerificationDisabled = true
	}

	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug("Email or phone number is required")
		return nil, fmt.Errorf(`email or phone number is required`)
	}
	log := log.WithFields(log.Fields{
		"email":        refs.StringValue(params.Email),
		"phone_number": refs.StringValue(params.PhoneNumber),
	})
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if isBasicAuthDisabled && isEmailLogin && !isEmailVerificationDisabled {
		log.Debug("Basic authentication is disabled.")
		return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileLogin && !isMobileVerificationDisabled {
		log.Debug("Mobile basic authentication is disabled.")
		return nil, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	var user *models.User
	if isEmailLogin {
		user, err = db.Provider.GetUserByEmail(ctx, email)
	} else {
		user, err = db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
	}
	if err != nil {
		log.Debug("Failed to get user: ", err)
		return nil, fmt.Errorf(`bad user credentials`)
	}
	hostname := parsers.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug("Failed to generate nonce: ", err)
		return nil, err
	}
	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}
	if isEmailLogin {
		redirectURI := ""
		// give higher preference to params redirect uri
		if strings.TrimSpace(refs.StringValue(params.RedirectURI)) != "" {
			redirectURI = refs.StringValue(params.RedirectURI)
		} else {
			redirectURI, err = memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyResetPasswordURL)
			if err != nil {
				log.Debug("ResetPasswordURL not found using default app url: ", err)
				redirectURI = hostname + "/app/reset-password"
				memorystore.Provider.UpdateEnvVariable(constants.EnvKeyResetPasswordURL, redirectURI)
			}
		}
		verificationToken, err := token.CreateVerificationToken(email, constants.VerificationTypeForgotPassword, hostname, nonceHash, redirectURI)
		if err != nil {
			log.Debug("Failed to create verification token", err)
			return nil, err
		}
		_, err = db.Provider.AddVerificationRequest(ctx, &models.VerificationRequest{
			Token:       verificationToken,
			Identifier:  constants.VerificationTypeForgotPassword,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURI,
		})
		if err != nil {
			log.Debug("Failed to add verification request", err)
			return nil, err
		}
		// execute it as go routine so that we can reduce the api latency
		go mailService.SendEmail([]string{email}, constants.VerificationTypeForgotPassword, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(),
			"verification_url": utils.GetForgotPasswordURL(verificationToken, redirectURI),
		})
		return &model.ForgotPasswordResponse{
			Message: `Please check your inbox! We have sent a password reset link.`,
		}, nil
	}
	if isMobileLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
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
		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return nil, err
		}
		cookie.SetMfaSession(gc, mfaSession)
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(otpData.Otp)
		if err := smsproviders.SendSMS(phoneNumber, smsBody.String()); err != nil {
			log.Debug("Failed to send sms: ", err)
			// continue
		}
		return &model.ForgotPasswordResponse{
			Message:                   "Please enter the OTP sent to your phone number and change your password.",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, nil
	}
	return nil, fmt.Errorf(`email or phone number verification needs to be enabled`)
}
