package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// UpdateUser is the method to update user details
// Permission: authorizer:admin
func (g *graphqlProvider) UpdateUser(ctx context.Context, params *model.UpdateUserInput) (*model.User, error) {
	log := g.Log.With().Str("func", "UpdateUser").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if params.ID == "" {
		log.Debug().Msg("user id is missing")
		return nil, fmt.Errorf("user_id is missing")
	}

	log = log.With().Str("user_id", params.ID).Logger()

	if params.GivenName == nil &&
		params.FamilyName == nil &&
		params.Picture == nil &&
		params.MiddleName == nil &&
		params.Nickname == nil &&
		params.Email == nil &&
		params.Birthdate == nil &&
		params.Gender == nil &&
		params.PhoneNumber == nil &&
		params.Roles == nil &&
		params.IsMultiFactorAuthEnabled == nil &&
		params.AppData == nil {
		log.Debug().Msg("please enter atleast one param to update")
		return nil, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := g.StorageProvider.GetUserByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByID")
		return nil, fmt.Errorf(`User not found`)
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
	// TODO
	// if params.PhoneNumber != nil && refs.StringValue(user.PhoneNumber) != refs.StringValue(params.PhoneNumber) {
	// 	// verify if phone number is unique
	// 	if _, err := g.StorageProvider.GetUserByPhoneNumber(ctx, strings.TrimSpace(refs.StringValue(params.PhoneNumber))); err == nil {
	// 		log.Debug().Msg("user with given phone number already exists")
	// 		return nil, errors.New("user with given phone number already exists")
	// 	}
	// 	user.PhoneNumber = params.PhoneNumber
	// }

	if params.Picture != nil && refs.StringValue(user.Picture) != refs.StringValue(params.Picture) {
		user.Picture = params.Picture
	}

	if params.AppData != nil {
		appDataString := ""
		appDataBytes, err := json.Marshal(params.AppData)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal app_data")
			return nil, errors.New("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}

	if params.IsMultiFactorAuthEnabled != nil && refs.BoolValue(user.IsMultiFactorAuthEnabled) != refs.BoolValue(params.IsMultiFactorAuthEnabled) {
		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
		if refs.BoolValue(params.IsMultiFactorAuthEnabled) {
			// Check if totp, email or sms is enabled
			isMailOTPEnvServiceDisabled := g.Config.DisableEmailOTP
			isTOTPEnvServiceDisabled := g.Config.DisableTOTPLogin
			isSMSOTPEnvServiceDisabled := g.Config.DisableSMSOTP
			// Initialize a flag to check if enabling Mail OTP is required
			if isMailOTPEnvServiceDisabled && isTOTPEnvServiceDisabled && isSMSOTPEnvServiceDisabled {
				log.Debug().Msg("cannot enable multi factor authentication as all mfa services are disabled")
				return nil, errors.New("cannot enable multi factor authentication as all mfa services are disabled")
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
	if params.PhoneNumberVerified != nil {
		if *params.PhoneNumberVerified {
			now := time.Now().Unix()
			user.PhoneNumberVerifiedAt = &now
		} else {
			user.PhoneNumberVerifiedAt = nil
		}

	}

	if params.Email != nil && refs.StringValue(user.Email) != refs.StringValue(params.Email) {
		// check if valid email
		if !validators.IsValidEmail(*params.Email) {
			log.Debug().Str("email", *params.Email).Msg("Invalid email address")
			return nil, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err = g.StorageProvider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug().Str("email", newEmail).Msg("User with email already exists")
			return nil, fmt.Errorf("user with this email address already exists")
		}

		go g.MemoryStoreProvider.DeleteAllUserSessions(user.ID)

		hostname := parsers.GetHost(gc)
		user.Email = &newEmail
		user.EmailVerifiedAt = nil
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate nonce")
			return nil, err
		}
		verificationType := constants.VerificationTypeUpdateEmail
		redirectURL := parsers.GetAppURL(gc)
		// newEmail, verificationType, hostname, nonceHash, redirectURL
		verificationToken, err := g.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			User:        user,
			Nonce:       nonceHash,
			HostName:    hostname,
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
		}, redirectURL, verificationType)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create verification token")
			return nil, err
		}
		_, err = g.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       newEmail,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, err
		}

		// exec it as go routine so that we can reduce the api latency
		go g.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(g.Config),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
		})

	}

	if params.PhoneNumber != nil && refs.StringValue(user.PhoneNumber) != refs.StringValue(params.PhoneNumber) {
		phone := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
		if len(phone) < 10 || len(phone) > 15 {
			log.Debug().Str("phone", phone).Msg("Invalid phone number")
			return nil, fmt.Errorf("invalid phone number")
		}
		// check if user with new phone number exists
		_, err = g.StorageProvider.GetUserByPhoneNumber(ctx, phone)
		// err = nil means user exists
		if err == nil {
			log.Debug().Str("phone", phone).Msg("User with phone number already exists")
			return nil, fmt.Errorf("user with this phone number already exists")
		}
		go g.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		user.PhoneNumber = &phone
		user.PhoneNumberVerifiedAt = nil
	}

	rolesToSave := ""
	if len(params.Roles) > 0 {
		currentRoles := strings.Split(user.Roles, ",")
		inputRoles := []string{}
		for _, item := range params.Roles {
			inputRoles = append(inputRoles, *item)
		}

		roles := g.Config.Roles
		protectedRoles := g.Config.ProtectedRoles

		if !validators.IsValidRoles(inputRoles, append([]string{}, append(roles, protectedRoles...)...)) {
			log.Debug().Msg("Invalid list of roles")
			return nil, fmt.Errorf("invalid list of roles")
		}

		if !validators.IsStringArrayEqual(inputRoles, currentRoles) {
			rolesToSave = strings.Join(inputRoles, ",")
		}

		go g.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
	}

	if rolesToSave != "" {
		user.Roles = rolesToSave
	}
	user, err = g.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateUser")
		return nil, err
	}

	return user.AsAPIUser(), nil
}
