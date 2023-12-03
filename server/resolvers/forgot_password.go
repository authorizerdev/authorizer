package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	mailService "github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ForgotPasswordResolver is a resolver for forgot password mutation
func ForgotPasswordResolver(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isEmailVerificationDisabled = true
	}

	isMobileBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isMobileBasicAuthDisabled = true
	}
	isMobileVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isMobileVerificationDisabled = true
	}

	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug("Email or phone number is required")
		return res, fmt.Errorf(`email or phone number is required`)
	}
	log := log.WithFields(log.Fields{
		"email":        refs.StringValue(params.Email),
		"phone_number": refs.StringValue(params.PhoneNumber),
	})
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if isBasicAuthDisabled && isEmailLogin && !isEmailVerificationDisabled {
		log.Debug("Basic authentication is disabled.")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileLogin && !isMobileVerificationDisabled {
		log.Debug("Mobile basic authentication is disabled.")
		return res, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	var user *models.User
	if isEmailLogin {
		user, err = db.Provider.GetUserByEmail(ctx, email)
	} else {
		user, err = db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
	}
	if err != nil {
		log.Debug("Failed to get user: ", err)
		return res, fmt.Errorf(`bad user credentials`)
	}
	hostname := parsers.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug("Failed to generate nonce: ", err)
		return res, err
	}
	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
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
			return res, err
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
			return res, err
		}
		// execute it as go routine so that we can reduce the api latency
		go mailService.SendEmail([]string{email}, constants.VerificationTypeForgotPassword, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(),
			"verification_url": utils.GetForgotPasswordURL(verificationToken, redirectURI),
		})
	}
	if isMobileLogin {
		// TODO: send sms
	}
	res = &model.Response{
		Message: `Please check your inbox! We have sent a password reset link.`,
	}

	return res, nil
}
