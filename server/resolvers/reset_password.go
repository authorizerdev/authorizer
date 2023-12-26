package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
)

// ResetPasswordResolver is a resolver for reset password mutation
func ResetPasswordResolver(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}
	verifyingToken := refs.StringValue(params.Token)
	otp := refs.StringValue(params.Otp)
	if verifyingToken == "" && otp == "" {
		log.Debug("Token or OTP is required")
		return res, fmt.Errorf(`token or otp is required`)
	}
	isTokenVerification := verifyingToken != ""
	isOtpVerification := otp != ""
	if isOtpVerification && refs.StringValue(params.PhoneNumber) == "" {
		log.Debug("Phone number is required")
		return res, fmt.Errorf(`phone number is required`)
	}
	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	isMobileBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	if isTokenVerification && isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isOtpVerification && isMobileBasicAuthDisabled {
		log.Debug("Mobile basic authentication is disabled")
		return res, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	email := ""
	phoneNumber := refs.StringValue(params.PhoneNumber)
	var user *models.User
	var verificationRequest *models.VerificationRequest
	var otpRequest *models.OTP
	if isTokenVerification {
		verificationRequest, err = db.Provider.GetVerificationRequestByToken(ctx, verifyingToken)
		if err != nil {
			log.Debug("Failed to get verification request: ", err)
			return res, fmt.Errorf(`invalid token`)
		}
		// verify if token exists in db
		hostname := parsers.GetHost(gc)
		claim, err := token.ParseJWTToken(verifyingToken)
		if err != nil {
			log.Debug("Failed to parse token: ", err)
			return res, fmt.Errorf(`invalid token`)
		}

		if ok, err := token.ValidateJWTClaims(claim, hostname, verificationRequest.Nonce, verificationRequest.Email); !ok || err != nil {
			log.Debug("Failed to validate jwt claims: ", err)
			return res, fmt.Errorf(`invalid token`)
		}
		email = claim["sub"].(string)
		user, err = db.Provider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug("Failed to get user: ", err)
			return res, err
		}
	}
	if isOtpVerification {
		mfaSession, err := cookie.GetMfaSession(gc)
		if err != nil {
			log.Debug("Failed to get otp request by email: ", err)
			return res, fmt.Errorf(`invalid session: %s`, err.Error())
		}
		// Get user by phone number
		user, err = db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug("Failed to get user by phone number: ", err)
			return res, fmt.Errorf(`user not found`)
		}
		if _, err := memorystore.Provider.GetMfaSession(user.ID, mfaSession); err != nil {
			log.Debug("Failed to get mfa session: ", err)
			return res, fmt.Errorf(`invalid session: %s`, err.Error())
		}
		otpRequest, err = db.Provider.GetOTPByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug("Failed to get otp request by phone number: ", err)
			return res, fmt.Errorf(`invalid otp`)
		}
		if otpRequest.Otp != otp {
			log.Debug("Failed to verify otp request: Incorrect value")
			return res, fmt.Errorf(`invalid otp`)
		}
	}
	if params.Password != params.ConfirmPassword {
		log.Debug("Passwords do not match")
		return res, fmt.Errorf(`passwords don't match`)
	}
	if err := validators.IsValidPassword(params.Password); err != nil {
		log.Debug("Invalid password")
		return res, err
	}
	log := log.WithFields(log.Fields{
		"email": email,
		"phone": phoneNumber,
	})
	password, _ := crypto.EncryptPassword(params.Password)
	user.Password = &password
	signupMethod := user.SignupMethods
	if !strings.Contains(signupMethod, constants.AuthRecipeMethodBasicAuth) && isTokenVerification {
		signupMethod = signupMethod + "," + constants.AuthRecipeMethodBasicAuth
		// helpful if user has not signed up with basic auth
		if user.EmailVerifiedAt == nil {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
		}
	}
	if !strings.Contains(signupMethod, constants.AuthRecipeMethodMobileOTP) && isOtpVerification {
		signupMethod = signupMethod + "," + constants.AuthRecipeMethodMobileOTP
		// helpful if user has not signed up with basic auth
		if user.PhoneNumberVerifiedAt == nil {
			now := time.Now().Unix()
			user.PhoneNumberVerifiedAt = &now
		}
	}
	user.SignupMethods = signupMethod
	isMFAEnforced, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyEnforceMultiFactorAuthentication)
	if err != nil {
		log.Debug("MFA service not enabled: ", err)
		isMFAEnforced = false
	}
	if isMFAEnforced {
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	}
	_, err = db.Provider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}
	if isTokenVerification {
		// delete from verification table
		err = db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug("Failed to delete verification request: ", err)
			return res, err
		}
	}
	if isOtpVerification {
		// delete from otp table
		err = db.Provider.DeleteOTP(ctx, otpRequest)
		if err != nil {
			log.Debug("Failed to delete otp request: ", err)
			return res, err
		}
	}
	res = &model.Response{
		Message: `Password updated successfully.`,
	}
	return res, nil
}
