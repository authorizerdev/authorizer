package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/audit"
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

// UpdateProfile updates the authenticated caller's profile. When the email
// changes (with email verification enabled) the current session is dropped and
// a verification email is sent; the cleared session cookies are returned as
// side-effects for the transport to apply.
// Transport-agnostic port of graphqlProvider.UpdateProfile.
//
// Permissions: authenticated:user
func (p *provider) UpdateProfile(ctx context.Context, meta RequestMetadata, params *model.UpdateProfileRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateProfile").Logger()
	side := &ResponseSideEffects{}

	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil {
		log.Debug().Err(err).Msg("Failed GetUserIDFromSessionOrAccessToken")
		return nil, nil, Unauthenticated(err.Error())
	}
	if tokenData == nil || tokenData.UserID == "" {
		return nil, nil, Unauthenticated("unauthorized")
	}

	// validate if all params are not empty
	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.OldPassword == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil && params.NewPassword == nil && params.ConfirmNewPassword == nil && params.IsMultiFactorAuthEnabled == nil && params.AppData == nil {
		log.Debug().Msg("All params are empty")
		return nil, nil, InvalidArgument("please enter at least one param to update")
	}
	log = log.With().Str("user_id", tokenData.UserID).Logger()
	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, nil, err
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
		if _, err := p.StorageProvider.GetUserByPhoneNumber(ctx, strings.TrimSpace(refs.StringValue(params.PhoneNumber))); err == nil {
			log.Debug().Msg("user with given phone number already exists")
			return nil, nil, InvalidArgument("user with given phone number already exists")
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
			return nil, nil, InvalidArgument("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}
	// Check if the user is trying to enable or disable multi-factor authentication (MFA)
	if params.IsMultiFactorAuthEnabled != nil && refs.BoolValue(user.IsMultiFactorAuthEnabled) != refs.BoolValue(params.IsMultiFactorAuthEnabled) {
		// Only gate the enable action; disabling is always allowed (subject to the
		// enforce check below). Uses the same availability rule as login-time gating.
		if refs.BoolValue(params.IsMultiFactorAuthEnabled) && !p.isMFAServiceAvailable() {
			log.Debug().Msg("Cannot enable mfa as no mfa method is available")
			return nil, nil, FailedPrecondition("cannot enable MFA: no MFA method is available on this server — ensure TOTP is enabled (do not set --disable-totp-login) or configure an email (SMTP) or SMS (Twilio) provider for OTP")
		}

		isMFAEnforced := p.Config.EnforceMFA
		if isMFAEnforced && !refs.BoolValue(params.IsMultiFactorAuthEnabled) {
			log.Debug().Msg("Cannot disable mfa service as it is enforced.")
			return nil, nil, FailedPrecondition("cannot disable multi factor authentication as it is enforced by organization")
		}

		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
	}

	isPasswordChanging := false
	if params.NewPassword != nil && params.ConfirmNewPassword == nil {
		log.Debug().Msg("confirm password is empty")
		return nil, nil, InvalidArgument("confirm password is required")
	}

	if params.ConfirmNewPassword != nil && params.NewPassword == nil {
		log.Debug().Msg("new password is empty")
		return nil, nil, InvalidArgument("new password is required")
	}

	if params.NewPassword != nil && params.ConfirmNewPassword != nil {
		isPasswordChanging = true
	}

	if isPasswordChanging && user.Password != nil && params.OldPassword == nil {
		log.Debug().Msg("old password is empty")
		return nil, nil, InvalidArgument("old password is required")
	}

	if isPasswordChanging && user.Password != nil && params.OldPassword != nil {
		if err = bcrypt.CompareHashAndPassword([]byte(refs.StringValue(user.Password)), []byte(refs.StringValue(params.OldPassword))); err != nil {
			log.Debug().Err(err).Msg("Failed to compare hash and old password")
			return nil, nil, InvalidArgument("incorrect old password")
		}
	}

	shouldAddBasicSignUpMethod := false
	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := p.Config.EnableMobileBasicAuthentication

	if params.NewPassword != nil && params.ConfirmNewPassword != nil {
		if !isBasicAuthEnabled && !isMobileBasicAuthEnabled {
			log.Debug().Msg("Cannot update password as basic authentication is disabled")
			return nil, nil, FailedPrecondition(`basic authentication is disabled for this instance`)
		}

		if refs.StringValue(params.ConfirmNewPassword) != refs.StringValue(params.NewPassword) {
			log.Debug().Msg("Failed to compare new password and confirm new password")
			return nil, nil, InvalidArgument(`password and confirm password does not match`)
		}

		if user.Password == nil || refs.StringValue(user.Password) == "" {
			shouldAddBasicSignUpMethod = true
		}

		if err := validators.IsValidPassword(refs.StringValue(params.NewPassword), !p.Config.EnableStrongPassword); err != nil {
			log.Debug().Msg("Invalid password")
			return nil, nil, InvalidArgument(err.Error())
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
			return nil, nil, InvalidArgument("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)

		// check if valid email
		if !validators.IsValidEmail(newEmail) {
			log.Debug().Str("new_email", newEmail).Msg("Failed to validate new email: ")
			return nil, nil, InvalidArgument("invalid new email address")
		}
		// check if user with new email exists
		_, err := p.StorageProvider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug().Str("new_email", newEmail).Msg("User with new email already exists")
			return nil, nil, InvalidArgument("user with this email address already exists")
		}

		go func() { _ = p.MemoryStoreProvider.DeleteAllUserSessions(user.ID) }()
		for _, c := range cookie.BuildDeleteSessionCookies(meta.HostURL, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
			side.AddCookie(c)
		}

		user.Email = &newEmail
		isEmailVerificationEnabled := p.Config.EnableEmailVerification
		if isEmailVerificationEnabled {
			hostname := meta.HostURL
			user.EmailVerifiedAt = nil
			hasEmailChanged = true
			// insert verification request
			_, nonceHash, err := utils.GenerateNonce()
			if err != nil {
				log.Debug().Err(err).Msg("Failed to generate nonce")
				return nil, nil, err
			}
			verificationType := constants.VerificationTypeUpdateEmail
			redirectURL := parsers.GetAppURLFromRequest(meta.Request)

			verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
				User:        user,
				HostName:    hostname,
				Nonce:       nonceHash,
				LoginMethod: constants.AuthRecipeMethodBasicAuth,
			}, redirectURL, verificationType)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to create verification token")
				return nil, nil, err
			}
			_, err = p.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
				Token:       verificationToken,
				Identifier:  verificationType,
				ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
				Email:       newEmail,
				Nonce:       nonceHash,
				RedirectURI: redirectURL,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Failed to add verification request")
				return nil, nil, err
			}

			// exec it as go routine so that we can reduce the api latency
			go func() {
				_ = p.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, verificationType, map[string]any{
					"user":             user.ToMap(),
					"organization":     utils.GetOrganization(p.Config),
					"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
				})
			}()

		}
	}
	_, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditProfileUpdatedEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	message := `Profile details updated successfully.`
	if hasEmailChanged {
		message += `For the email change we have sent new verification email, please verify and continue`
	}

	return &model.Response{
		Message: message,
	}, side, nil
}
