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
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
	"github.com/authorizerdev/authorizer/server/smsproviders"
)

// ForgotPasswordResolver is a resolver for forgot password mutation
func ForgotPasswordResolver(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	disablePhoneVerification, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)

	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}

	if isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	mobile := strings.TrimSpace(params.EmailOrPhone)

	if !validators.IsValidEmail(params.EmailOrPhone) && len(mobile) < 10 {
		log.Debug("Invalid email or phone: ", params.EmailOrPhone)
		return res, fmt.Errorf("invalid email or phone")
	}

	if validators.IsValidEmail(params.EmailOrPhone) {
		
		params.EmailOrPhone = strings.ToLower(params.EmailOrPhone)

		log := log.WithFields(log.Fields{
			"email": params.EmailOrPhone,
		})
		user, err := db.Provider.GetUserByEmail(ctx, params.EmailOrPhone)
		if err != nil {
			log.Debug("User not found: ", err)
			return res, fmt.Errorf(`user with this email not found`)
		}

		hostname := parsers.GetHost(gc)
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug("Failed to generate nonce: ", err)
			return res, err
		}

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

		verificationToken, err := token.CreateVerificationToken(params.EmailOrPhone, constants.VerificationTypeForgotPassword, hostname, nonceHash, redirectURI)
		if err != nil {
			log.Debug("Failed to create verification token", err)
			return res, err
		}
		_, err = db.Provider.AddVerificationRequest(ctx, models.VerificationRequest{
			Token:       verificationToken,
			Identifier:  constants.VerificationTypeForgotPassword,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       params.EmailOrPhone,
			Nonce:       nonceHash,
			RedirectURI: redirectURI,
		})
		if err != nil {
			log.Debug("Failed to add verification request", err)
			return res, err
		}

		// execute it as go routine so that we can reduce the api latency
		go email.SendEmail([]string{params.EmailOrPhone}, constants.VerificationTypeForgotPassword, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(),
			"verification_url": utils.GetForgotPasswordURL(verificationToken, redirectURI),
		})

		res = &model.Response{
			Message: `Please check your inbox! We have sent a password reset link.`,
		}

    	return res, nil
	}

    if !disablePhoneVerification && len(mobile) > 9 {
	
		if _, err := db.Provider.GetUserByPhoneNumber(ctx, refs.StringValue(&params.EmailOrPhone)); err != nil {
			return res, fmt.Errorf("user with given phone number does not exist")
		}
		
		duration, _ := time.ParseDuration("10m")
		smsCode := utils.GenerateOTP()
	
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(smsCode)

		go func() {
			_, err = db.Provider.UpsertSMSRequest(ctx, &models.SMSVerificationRequest{
				PhoneNumber:     params.EmailOrPhone,
				Code:   	     smsCode,
				CodeExpiresAt:   time.Now().Add(duration).Unix(),
			})

			if err != nil {
				log.Debug("Failed to upsert sms otp: ", err)
				return
			}
			
			err = smsproviders.SendSMS(params.EmailOrPhone, smsBody.String())
			if err != nil {
				log.Debug("Failed to send sms: ", err)
				return
			}
		}()

		res = &model.Response{
			Message: `verification code has been sent to your phone`,
		}
	
		return res, nil
	}

	return res, nil

}
