package resolvers

import (
	"context"
	"encoding/json"
	"errors"
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
)

// UpdateUserResolver is a resolver for update user mutation
// This is admin only mutation
func UpdateUserResolver(ctx context.Context, params model.UpdateUserInput) (*model.User, error) {
	var res *model.User

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return res, fmt.Errorf("unauthorized")
	}

	if params.ID == "" {
		log.Debug("UserID is empty")
		return res, fmt.Errorf("User ID is required")
	}

	log := log.WithFields(log.Fields{
		"user_id": params.ID,
	})

	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil && params.Roles == nil && params.IsMultiFactorAuthEnabled == nil {
		log.Debug("No params to update")
		return res, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := db.Provider.GetUserByID(ctx, params.ID)
	if err != nil {
		log.Debug("Failed to get user by id: ", err)
		return res, fmt.Errorf(`User not found`)
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

	if params.IsMultiFactorAuthEnabled != nil && refs.BoolValue(user.IsMultiFactorAuthEnabled) != refs.BoolValue(params.IsMultiFactorAuthEnabled) {
		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
		if refs.BoolValue(params.IsMultiFactorAuthEnabled) {
			isEnvServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
			isMailOTPEnvServiceEnabled, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMailOTPLogin)
			isTOTPEnvServiceEnabled, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableTOTPLogin)
			checkMailOTP := !isEnvServiceEnabled && !isTOTPEnvServiceEnabled && isMailOTPEnvServiceEnabled
			if err != nil || !checkMailOTP {
				log.Debug("Email service not enabled:")
				return nil, errors.New("email service not enabled, so cannot enable multi factor authentication")
			}
		}
	}

	if params.EmailVerified != nil {
		if *params.EmailVerified {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
		} else {
			user.EmailVerifiedAt = nil
		}
	}

	if params.Email != nil && user.Email != *params.Email {
		// check if valid email
		if !validators.IsValidEmail(*params.Email) {
			log.Debug("Invalid email: ", *params.Email)
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err = db.Provider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug("User with email already exists: ", newEmail)
			return res, fmt.Errorf("user with this email address already exists")
		}

		go memorystore.Provider.DeleteAllUserSessions(user.ID)

		hostname := parsers.GetHost(gc)
		user.Email = newEmail
		user.EmailVerifiedAt = nil
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
		go email.SendEmail([]string{user.Email}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
		})

	}

	rolesToSave := ""
	if params.Roles != nil && len(params.Roles) > 0 {
		currentRoles := strings.Split(user.Roles, ",")
		inputRoles := []string{}
		for _, item := range params.Roles {
			inputRoles = append(inputRoles, *item)
		}

		rolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyRoles)
		roles := []string{}
		if err != nil {
			log.Debug("Error getting roles: ", err)
			rolesString = ""
		} else {
			roles = strings.Split(rolesString, ",")
		}
		protectedRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyProtectedRoles)
		protectedRoles := []string{}
		if err != nil {
			log.Debug("Error getting protected roles: ", err)
			protectedRolesString = ""
		} else {
			protectedRoles = strings.Split(protectedRolesString, ",")
		}

		if !validators.IsValidRoles(inputRoles, append([]string{}, append(roles, protectedRoles...)...)) {
			log.Debug("Invalid roles: ", params.Roles)
			return res, fmt.Errorf("invalid list of roles")
		}

		if !validators.IsStringArrayEqual(inputRoles, currentRoles) {
			rolesToSave = strings.Join(inputRoles, ",")
		}

		go memorystore.Provider.DeleteAllUserSessions(user.ID)
	}

	if rolesToSave != "" {
		user.Roles = rolesToSave
	}

	user, err = db.Provider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}

	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
	res = &model.User{
		ID:         params.ID,
		Email:      user.Email,
		Picture:    user.Picture,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
		Roles:      strings.Split(user.Roles, ","),
		CreatedAt:  &createdAt,
		UpdatedAt:  &updatedAt,
	}
	return res, nil
}
