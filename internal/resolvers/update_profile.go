package resolvers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/models/db"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// UpdateProfileResolver is resolver for update profile mutation
func UpdateProfileResolver(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}
	tokenData, err := token.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug("Failed GetUserIDFromSessionOrAccessToken: ", err)
		return res, err
	}

	// validate if all params are not empty
	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.OldPassword == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil && params.NewPassword == nil && params.ConfirmNewPassword == nil && params.IsMultiFactorAuthEnabled == nil && params.AppData == nil {
		log.Debug("All params are empty")
		return res, fmt.Errorf("please enter at least one param to update")
	}
	log := log.WithFields(log.Fields{
		"user_id": tokenData.UserID,
	})
	user, err := db.Provider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug("Failed to get user by id: ", err)
		return res, err
	}

	if params.GivenName != nil && refs.StringValue(user.GivenName) != refs.StringValue(params.GivenName) {
		user.GivenName = params.GivenName
	}

	if params.FamilyName != nil && refs.StringValue(user.FamilyName) != refs.StringValue(params.FamilyName) {
		user.FamilyName = params.FamilyName
	}

	if params.MiddleName != nil && refs.StringValue(user.MiddleName) != refs.StringValue(params.MiddleName) {
		user.MiddleName = params.MiddleName
	}

	if params.Nickname != nil && refs.StringValue(user.Nickname) != refs.StringValue(params.Nickname) {
		user.Nickname = params.Nickname
	}

	if params.Birthdate != nil && refs.StringValue(user.Birthdate) != refs.StringValue(params.Birthdate) {
		user.Birthdate = params.Birthdate
	}

	if params.Gender != nil && refs.StringValue(user.Gender) != refs.StringValue(params.Gender) {
		user.Gender = params.Gender
	}

	if params.PhoneNumber != nil && refs.StringValue(user.PhoneNumber) != refs.StringValue(params.PhoneNumber) {
		// verify if phone number is unique
		if _, err := db.Provider.GetUserByPhoneNumber(ctx, strings.TrimSpace(refs.StringValue(params.PhoneNumber))); err == nil {
			log.Debug("user with given phone number already exists")
			return nil, errors.New("user with given phone number already exists")
		}
		user.PhoneNumber = params.PhoneNumber
	}

	if params.Picture != nil && refs.StringValue(user.Picture) != refs.StringValue(params.Picture) {
		user.Picture = params.Picture
	}
	if params.AppData != nil {
		appDataString := ""
		appDataBytes, err := json.Marshal(params.AppData)
		if err != nil {
			log.Debug("failed to marshall source app_data: ", err)
			return nil, errors.New("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}
	// Check if the user is trying to enable or disable multi-factor authentication (MFA)
	if params.IsMultiFactorAuthEnabled != nil && refs.BoolValue(user.IsMultiFactorAuthEnabled) != refs.BoolValue(params.IsMultiFactorAuthEnabled) {
		// Check if totp, email or sms is enabled
		isMailOTPEnvServiceDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMailOTPLogin)
		if err != nil {
			log.Debug("Error getting mail otp disabled: ", err)
			isMailOTPEnvServiceDisabled = false
		}
		isTOTPEnvServiceDisabled, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableTOTPLogin)
		if err != nil {
			log.Debug("Error getting totp disabled: ", err)
			isTOTPEnvServiceDisabled = false
		}
		isSMSOTPEnvServiceDisabled, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
		if err != nil {
			log.Debug("Error getting sms otp disabled: ", err)
			isSMSOTPEnvServiceDisabled = false
		}
		// Initialize a flag to check if enabling Mail OTP is required
		if isMailOTPEnvServiceDisabled && isTOTPEnvServiceDisabled && isSMSOTPEnvServiceDisabled {
			log.Debug("Cannot enable mfa service as all mfa services are disabled")
			return nil, errors.New("cannot enable multi factor authentication as all mfa services are disabled")
		}

		isMFAEnforced, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyEnforceMultiFactorAuthentication)
		if err != nil {
			log.Debug("MFA service not enabled: ", err)
			isMFAEnforced = false
		}

		if isMFAEnforced && !refs.BoolValue(params.IsMultiFactorAuthEnabled) {
			log.Debug("Cannot disable mfa service as it is enforced:")
			return nil, errors.New("cannot disable multi factor authentication as it is enforced by organization")
		}

		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
	}

	isPasswordChanging := false
	if params.NewPassword != nil && params.ConfirmNewPassword == nil {
		isPasswordChanging = true
		log.Debug("confirm password is empty")
		return res, fmt.Errorf("confirm password is required")
	}

	if params.ConfirmNewPassword != nil && params.NewPassword == nil {
		isPasswordChanging = true
		log.Debug("new password is empty")
		return res, fmt.Errorf("new password is required")
	}

	if params.NewPassword != nil && params.ConfirmNewPassword != nil {
		isPasswordChanging = true
	}

	if isPasswordChanging && user.Password != nil && params.OldPassword == nil {
		log.Debug("old password is empty")
		return res, fmt.Errorf("old password is required")
	}

	if isPasswordChanging && user.Password != nil && params.OldPassword != nil {
		if err = bcrypt.CompareHashAndPassword([]byte(refs.StringValue(user.Password)), []byte(refs.StringValue(params.OldPassword))); err != nil {
			log.Debug("Failed to compare hash and old password: ", err)
			return res, fmt.Errorf("incorrect old password")
		}
	}

	shouldAddBasicSignUpMethod := false
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

	if params.NewPassword != nil && params.ConfirmNewPassword != nil {
		if isBasicAuthDisabled || isMobileBasicAuthDisabled {
			log.Debug("Cannot update password as basic authentication is disabled")
			return res, fmt.Errorf(`basic authentication is disabled for this instance`)
		}

		if refs.StringValue(params.ConfirmNewPassword) != refs.StringValue(params.NewPassword) {
			log.Debug("Failed to compare new password and confirm new password")
			return res, fmt.Errorf(`password and confirm password does not match`)
		}

		if user.Password == nil || refs.StringValue(user.Password) == "" {
			shouldAddBasicSignUpMethod = true
		}

		if err := validators.IsValidPassword(refs.StringValue(params.NewPassword)); err != nil {
			log.Debug("Invalid password")
			return res, err
		}

		password, _ := crypto.EncryptPassword(refs.StringValue(params.NewPassword))
		user.Password = &password

		if shouldAddBasicSignUpMethod {
			user.SignupMethods = user.SignupMethods + "," + constants.AuthRecipeMethodBasicAuth
		}
	}

	hasEmailChanged := false

	if params.Email != nil && refs.StringValue(user.Email) != refs.StringValue(params.Email) {
		// check if valid email
		if !validators.IsValidEmail(*params.Email) {
			log.Debug("Failed to validate email: ", refs.StringValue(params.Email))
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)

		// check if valid email
		if !validators.IsValidEmail(newEmail) {
			log.Debug("Failed to validate new email: ", newEmail)
			return res, fmt.Errorf("invalid new email address")
		}
		// check if user with new email exists
		_, err := db.Provider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug("Failed to get user by email: ", newEmail)
			return res, fmt.Errorf("user with this email address already exists")
		}

		go memorystore.Provider.DeleteAllUserSessions(user.ID)
		go cookie.DeleteSession(gc)

		user.Email = &newEmail
		isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
		if err != nil {
			log.Debug("Failed to get disable email verification env variable: ", err)
			return res, err
		}
		if !isEmailVerificationDisabled {
			hostname := parsers.GetHost(gc)
			user.EmailVerifiedAt = nil
			hasEmailChanged = true
			// insert verification request
			_, nonceHash, err := utils.GenerateNonce()
			if err != nil {
				log.Debug("Failed to generate nonce: ", err)
				return res, err
			}
			verificationType := constants.VerificationTypeUpdateEmail
			redirectURL := parsers.GetAppURL(gc)
			verificationToken, err := token.CreateVerificationToken(newEmail, verificationType, hostname, nonceHash, redirectURL)
			if err != nil {
				log.Debug("Failed to create verification token: ", err)
				return res, err
			}
			_, err = db.Provider.AddVerificationRequest(ctx, &models.VerificationRequest{
				Token:       verificationToken,
				Identifier:  verificationType,
				ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
				Email:       newEmail,
				Nonce:       nonceHash,
				RedirectURI: redirectURL,
			})
			if err != nil {
				log.Debug("Failed to add verification request: ", err)
				return res, err
			}

			// exec it as go routine so that we can reduce the api latency
			go email.SendEmail([]string{refs.StringValue(user.Email)}, verificationType, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
			})

		}
	}
	_, err = db.Provider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}
	message := `Profile details updated successfully.`
	if hasEmailChanged {
		message += `For the email change we have sent new verification email, please verify and continue`
	}
	res = &model.Response{
		Message: message,
	}

	return res, nil
}
