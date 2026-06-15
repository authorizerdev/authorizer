package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// Users returns a paginated list of all users. Requires super-admin auth.
// Logic migrated from internal/graphql/users.go.
func (p *provider) Users(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.Users, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Users").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	pagination := utils.GetPagination(params)
	res, pagination, err := p.StorageProvider.ListUsers(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListUsers")
		return nil, nil, err
	}
	resItems := make([]*model.User, len(res))
	for i, user := range res {
		resItems[i] = user.AsAPIUser()
	}
	return &model.Users{
		Pagination: pagination,
		Users:      resItems,
	}, nil, nil
}

// User returns a single user by id or email. Requires super-admin auth.
// Logic migrated from internal/graphql/user.go.
func (p *provider) User(ctx context.Context, meta RequestMetadata, params *model.GetUserRequest) (*model.User, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "User").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}
	// Try getting user by ID.
	if params.ID != nil && strings.Trim(*params.ID, " ") != "" {
		res, err := p.StorageProvider.GetUserByID(ctx, *params.ID)
		if err != nil {
			log.Debug().Err(err).Msg("failed GetUserByID")
			return nil, nil, err
		}
		return res.AsAPIUser(), nil, nil
	}
	// Try getting user by email.
	if params.Email != nil && strings.Trim(*params.Email, " ") != "" {
		res, err := p.StorageProvider.GetUserByEmail(ctx, *params.Email)
		if err != nil {
			log.Debug().Err(err).Msg("failed GetUserByEmail")
			return nil, nil, err
		}
		return res.AsAPIUser(), nil, nil
	}
	// Return error if no params are provided.
	return nil, nil, fmt.Errorf("invalid params, user id or email is required")
}

// UpdateUser updates a user's profile, roles, MFA, or verification state and
// returns the updated user. Requires super-admin auth. Logic migrated from
// internal/graphql/update_user.go.
func (p *provider) UpdateUser(ctx context.Context, meta RequestMetadata, params *model.UpdateUserRequest) (*model.User, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateUser").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("user id is missing")
		return nil, nil, fmt.Errorf("user_id is missing")
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
		return nil, nil, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := p.StorageProvider.GetUserByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByID")
		return nil, nil, fmt.Errorf(`User not found`)
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
	if params.Picture != nil && refs.StringValue(user.Picture) != refs.StringValue(params.Picture) {
		user.Picture = params.Picture
	}

	if params.AppData != nil {
		appDataString := ""
		appDataBytes, err := json.Marshal(params.AppData)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal app_data")
			return nil, nil, errors.New("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}

	if params.IsMultiFactorAuthEnabled != nil && refs.BoolValue(user.IsMultiFactorAuthEnabled) != refs.BoolValue(params.IsMultiFactorAuthEnabled) {
		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
		if refs.BoolValue(params.IsMultiFactorAuthEnabled) {
			// Check if totp, email or sms is enabled.
			isMailOTPEnvServiceEnabled := p.Config.EnableEmailOTP
			isTOTPEnvServiceEnabled := p.Config.EnableTOTPLogin
			isSMSOTPEnvServiceEnabled := p.Config.EnableSMSOTP
			// Initialize a flag to check if enabling Mail OTP is required.
			if !isMailOTPEnvServiceEnabled && !isTOTPEnvServiceEnabled && !isSMSOTPEnvServiceEnabled {
				log.Debug().Msg("cannot enable multi factor authentication as all mfa services are disabled")
				return nil, nil, errors.New("cannot enable multi factor authentication as all mfa services are disabled")
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
			return nil, nil, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err = p.StorageProvider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug().Str("email", newEmail).Msg("User with email already exists")
			return nil, nil, fmt.Errorf("user with this email address already exists")
		}

		go p.MemoryStoreProvider.DeleteAllUserSessions(user.ID)

		// gin-shim: parsers.GetHost / GetAppURL still take a *gin.Context.
		gc := &gin.Context{Request: meta.Request}
		hostname := parsers.GetHost(gc)
		user.Email = &newEmail
		user.EmailVerifiedAt = nil
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate nonce")
			return nil, nil, err
		}
		verificationType := constants.VerificationTypeUpdateEmail
		redirectURL := parsers.GetAppURL(gc)
		// newEmail, verificationType, hostname, nonceHash, redirectURL
		verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			User:        user,
			Nonce:       nonceHash,
			HostName:    hostname,
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
		go p.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(p.Config),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
		})
	}

	if params.PhoneNumber != nil && refs.StringValue(user.PhoneNumber) != refs.StringValue(params.PhoneNumber) {
		phone := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
		if len(phone) < 10 || len(phone) > 15 {
			log.Debug().Str("phone", phone).Msg("Invalid phone number")
			return nil, nil, fmt.Errorf("invalid phone number")
		}
		// check if user with new phone number exists
		_, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phone)
		// err = nil means user exists
		if err == nil {
			log.Debug().Str("phone", phone).Msg("User with phone number already exists")
			return nil, nil, fmt.Errorf("user with this phone number already exists")
		}
		go p.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
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

		roles := p.Config.Roles
		protectedRoles := p.Config.ProtectedRoles

		if !validators.IsValidRoles(inputRoles, append([]string{}, append(roles, protectedRoles...)...)) {
			log.Debug().Msg("Invalid list of roles")
			return nil, nil, fmt.Errorf("invalid list of roles")
		}

		if !validators.IsStringArrayEqual(inputRoles, currentRoles) {
			rolesToSave = strings.Join(inputRoles, ",")
		}

		go p.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
	}

	if rolesToSave != "" {
		user.Roles = rolesToSave
	}
	user, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateUser")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminUserUpdatedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return user.AsAPIUser(), nil, nil
}

// DeleteUser deletes a user (and associated OTP/verification data) by email.
// Requires super-admin auth. Logic migrated from internal/graphql/delete_user.go.
func (p *provider) DeleteUser(ctx context.Context, meta RequestMetadata, params *model.DeleteUserRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteUser").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	log = log.With().Str("email", params.Email).Logger()
	user, err := p.StorageProvider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by email")
		return nil, nil, err
	}

	err = p.StorageProvider.DeleteUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete user")
		return nil, nil, err
	}

	res := &model.Response{
		Message: `user deleted successfully`,
	}

	go func() {
		// delete otp for given email
		otp, err := p.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(user.Email))
		if err != nil {
			log.Debug().Err(err).Msg("No OTP found for email")
			// continue
		} else {
			err := p.StorageProvider.DeleteOTP(ctx, otp)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to delete otp for given email")
				// continue
			}
		}

		// delete otp for given phone number
		otp, err = p.StorageProvider.GetOTPByPhoneNumber(ctx, refs.StringValue(user.PhoneNumber))
		if err != nil {
			log.Debug().Err(err).Msg("No OTP found for phone number")
			// continue
		} else {
			err := p.StorageProvider.DeleteOTP(ctx, otp)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to delete otp for given phone number")
				// continue
			}
		}

		// delete verification requests for given email
		for _, vt := range constants.VerificationTypes {
			verificationRequest, err := p.StorageProvider.GetVerificationRequestByEmail(ctx, refs.StringValue(user.Email), vt)
			if err != nil {
				log.Debug().Err(err).Msg("No verification request found for email")
				// continue
			} else {
				err := p.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to delete verification request for given email")
					// continue
				}
			}
		}

		p.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		p.EventsProvider.RegisterEvent(ctx, constants.UserDeletedWebhookEvent, "", user)
	}()
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminUserDeletedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return res, nil, nil
}

// VerificationRequests returns a paginated list of pending verification
// requests. Requires super-admin auth. Logic migrated from
// internal/graphql/verification_requests.go.
func (p *provider) VerificationRequests(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.VerificationRequests, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "VerificationRequests").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	pagination := utils.GetPagination(params)
	requests, pagination, err := p.StorageProvider.ListVerificationRequests(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListVerificationRequests")
		return nil, nil, err
	}

	res := make([]*model.VerificationRequest, len(requests))
	for i, request := range requests {
		res[i] = request.AsAPIVerificationRequest()
	}

	return &model.VerificationRequests{
		Pagination:           pagination,
		VerificationRequests: res,
	}, nil, nil
}
