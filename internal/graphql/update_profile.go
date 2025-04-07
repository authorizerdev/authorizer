package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// UpdateProfile is the method to update profile
// Permissions: authenticated:user
func (g *graphqlProvider) UpdateProfile(ctx context.Context, params *model.UpdateProfileInput) (*model.Response, error) {
	log := g.Log.With().Str("func", "UpdateProfile").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	tokenData, err := g.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed GetUserIDFromSessionOrAccessToken")
		return nil, err
	}

	// validate if all params are not empty
	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.OldPassword == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil && params.NewPassword == nil && params.ConfirmNewPassword == nil && params.IsMultiFactorAuthEnabled == nil && params.AppData == nil {
		log.Debug().Msg("All params are empty")
		return nil, fmt.Errorf("please enter at least one param to update")
	}
	log = log.With().Str("user_id", tokenData.UserID).Logger()
	user, err := g.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, err
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
		if _, err := g.StorageProvider.GetUserByPhoneNumber(ctx, strings.TrimSpace(refs.StringValue(params.PhoneNumber))); err == nil {
			log.Debug().Msg("user with given phone number already exists")
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
			log.Debug().Err(err).Msg("failed to marshall source app_data")
			return nil, errors.New("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}
	// Check if the user is trying to enable or disable multi-factor authentication (MFA)
	if params.IsMultiFactorAuthEnabled != nil && refs.BoolValue(user.IsMultiFactorAuthEnabled) != refs.BoolValue(params.IsMultiFactorAuthEnabled) {
		// Check if totp, email or sms is enabled
		isMailOTPEnvServiceDisabled := g.Config.DisableEmailOTP
		isTOTPEnvServiceDisabled := g.Config.DisableTOTPLogin
		isSMSOTPEnvServiceDisabled := g.Config.DisableSMSOTP
		// Initialize a flag to check if enabling Mail OTP is required
		if isMailOTPEnvServiceDisabled && isTOTPEnvServiceDisabled && isSMSOTPEnvServiceDisabled {
			log.Debug().Msg("Cannot enable mfa service as all mfa services are disabled")
			return nil, errors.New("cannot enable multi factor authentication as all mfa services are disabled")
		}

		isMFAEnforced := g.Config.EnforceMFA
		if isMFAEnforced && !refs.BoolValue(params.IsMultiFactorAuthEnabled) {
			log.Debug().Msg("Cannot disable mfa service as it is enforced.")
			return nil, errors.New("cannot disable multi factor authentication as it is enforced by organization")
		}

		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
	}

	isPasswordChanging := false
	if params.NewPassword != nil && params.ConfirmNewPassword == nil {
		isPasswordChanging = true
		log.Debug().Msg("confirm password is empty")
		return nil, fmt.Errorf("confirm password is required")
	}

	if params.ConfirmNewPassword != nil && params.NewPassword == nil {
		isPasswordChanging = true
		log.Debug().Msg("new password is empty")
		return nil, fmt.Errorf("new password is required")
	}

	if params.NewPassword != nil && params.ConfirmNewPassword != nil {
		isPasswordChanging = true
	}

	if isPasswordChanging && user.Password != nil && params.OldPassword == nil {
		log.Debug().Msg("old password is empty")
		return nil, fmt.Errorf("old password is required")
	}

	if isPasswordChanging && user.Password != nil && params.OldPassword != nil {
		if err = bcrypt.CompareHashAndPassword([]byte(refs.StringValue(user.Password)), []byte(refs.StringValue(params.OldPassword))); err != nil {
			log.Debug().Err(err).Msg("Failed to compare hash and old password")
			return nil, fmt.Errorf("incorrect old password")
		}
	}

	shouldAddBasicSignUpMethod := false
	isBasicAuthDisabled := g.Config.DisableBasicAuthentication
	isMobileBasicAuthDisabled := g.Config.DisableMobileBasicAuthentication

	if params.NewPassword != nil && params.ConfirmNewPassword != nil {
		if isBasicAuthDisabled || isMobileBasicAuthDisabled {
			log.Debug().Msg("Cannot update password as basic authentication is disabled")
			return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
		}

		if refs.StringValue(params.ConfirmNewPassword) != refs.StringValue(params.NewPassword) {
			log.Debug().Msg("Failed to compare new password and confirm new password")
			return nil, fmt.Errorf(`password and confirm password does not match`)
		}

		if user.Password == nil || refs.StringValue(user.Password) == "" {
			shouldAddBasicSignUpMethod = true
		}

		if err := validators.IsValidPassword(refs.StringValue(params.NewPassword), g.Config.DisableStrongPassword); err != nil {
			log.Debug().Msg("Invalid password")
			return nil, err
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
			log.Debug().Str("email", refs.StringValue(params.Email)).Msg("Failed to validate email")
			return nil, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)

		// check if valid email
		if !validators.IsValidEmail(newEmail) {
			log.Debug().Str("new_email", newEmail).Msg("Failed to validate new email: ")
			return nil, fmt.Errorf("invalid new email address")
		}
		// check if user with new email exists
		_, err := g.StorageProvider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug().Str("new_email", newEmail).Msg("User with new email already exists")
			return nil, fmt.Errorf("user with this email address already exists")
		}

		go g.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		go cookie.DeleteSession(gc, g.Config.AppCookieSecure)

		user.Email = &newEmail
		isEmailVerificationDisabled := g.Config.DisableEmailVerification
		if !isEmailVerificationDisabled {
			hostname := parsers.GetHost(gc)
			user.EmailVerifiedAt = nil
			hasEmailChanged = true
			// insert verification request
			_, nonceHash, err := utils.GenerateNonce()
			if err != nil {
				log.Debug().Err(err).Msg("Failed to generate nonce")
				return nil, err
			}
			verificationType := constants.VerificationTypeUpdateEmail
			redirectURL := parsers.GetAppURL(gc)

			verificationToken, err := g.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
				User:        user,
				HostName:    hostname,
				Nonce:       nonceHash,
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
			go g.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, verificationType, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(g.Config),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
			})

		}
	}
	_, err = g.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, err
	}
	message := `Profile details updated successfully.`
	if hasEmailChanged {
		message += `For the email change we have sent new verification email, please verify and continue`
	}

	return &model.Response{
		Message: message,
	}, nil
}
