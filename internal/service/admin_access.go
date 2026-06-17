package service

import (
	"context"
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

// RevokeAccess revokes a user's access by setting the revoked timestamp,
// killing their active sessions, and firing the access-revoked webhook.
// Requires super-admin auth. Logic migrated from internal/graphql/revoke_access.go.
func (p *provider) RevokeAccess(ctx context.Context, meta RequestMetadata, params *model.UpdateAccessRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RevokeAccess").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	log = log.With().Str("user_id", params.UserID).Logger()
	user, err := p.StorageProvider.GetUserByID(ctx, params.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, nil, err
	}

	now := time.Now().Unix()
	user.RevokedTimestamp = &now

	user, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}

	go func() {
		ctx := context.WithoutCancel(ctx)
		_ = p.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		_ = p.EventsProvider.RegisterEvent(ctx, constants.UserAccessRevokedWebhookEvent, "", user)
	}()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminAccessRevokedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: `user access revoked successfully`,
	}, nil, nil
}

// EnableAccess re-enables a previously revoked user by clearing the revoked
// timestamp and fires the access-enabled webhook. Requires super-admin auth.
// Logic migrated from internal/graphql/enable_access.go.
func (p *provider) EnableAccess(ctx context.Context, meta RequestMetadata, params *model.UpdateAccessRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "EnableAccess").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.UserID == "" {
		return nil, nil, fmt.Errorf("user ID is missing")
	}

	log = log.With().Str("user_id", params.UserID).Logger()

	user, err := p.StorageProvider.GetUserByID(ctx, params.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by ID")
		return nil, nil, err
	}

	user.RevokedTimestamp = nil

	user, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}
	go func() {
		ctx := context.WithoutCancel(ctx)
		_ = p.EventsProvider.RegisterEvent(ctx, constants.UserAccessEnabledWebhookEvent, "", user)
	}()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminAccessEnabledEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: `user access enabled successfully`,
	}, nil, nil
}

// InviteMembers creates accounts for the supplied emails that do not yet exist
// and sends each an invite (magic-link or setup-password) email. Requires
// super-admin auth and a configured email service. Logic migrated from
// internal/graphql/invite_members.go.
func (p *provider) InviteMembers(ctx context.Context, meta RequestMetadata, params *model.InviteMemberRequest) (*model.InviteMembersResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "InviteMembers").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	// this feature is only allowed if email server is configured
	if !p.Config.IsEmailServiceEnabled {
		log.Debug().Msg("Email sending is disabled")
		return nil, nil, errors.New("email sending is disabled")
	}

	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isMagicLinkLoginEnabled := p.Config.EnableMagicLinkLogin
	if !isBasicAuthEnabled && !isMagicLinkLoginEnabled {
		log.Debug().Msg("Either basic authentication or magic link login is required")
		return nil, nil, errors.New("either basic authentication or magic link login is required")
	}

	// filter valid emails
	emails := []string{}
	for _, email := range params.Emails {
		if validators.IsValidEmail(email) {
			emails = append(emails, email)
		}
	}

	if len(emails) == 0 {
		log.Debug().Msg("No valid emails found")
		return nil, nil, errors.New("no valid emails found")
	}

	// TODO: optimise to use like query instead of looping through emails and getting user individually
	// for each emails check if emails exists in db
	newEmails := []string{}
	for _, email := range emails {
		_, err := p.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Msgf("User with %s email does not exist", email)
			newEmails = append(newEmails, email)
		} else {
			log.Debug().Msgf("User with %s email already exists", email)
		}
	}

	if len(newEmails) == 0 {
		log.Debug().Msg("All emails already exist")
		return nil, nil, errors.New("all emails already exist")
	}

	// gin-shim: parsers.GetHost / GetAppURL still take a *gin.Context.
	gc := &gin.Context{Request: meta.Request}

	// invite new emails
	for _, email := range newEmails {
		user := &schemas.User{
			Email: refs.NewStringRef(email),
			Roles: strings.Join(p.Config.DefaultRoles, ","),
		}
		hostname := parsers.GetHost(gc)
		verifyEmailURL := hostname + "/verify_email"
		appURL := parsers.GetAppURL(gc)

		redirectURL := appURL
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
			if !validators.IsValidRedirectURI(redirectURL, p.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				return nil, nil, fmt.Errorf("invalid redirect URI")
			}
		}

		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			return nil, nil, err
		}

		verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
			Nonce:       nonceHash,
			HostName:    hostname,
			User:        user,
		}, redirectURL, constants.VerificationTypeInviteMember)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create verification token")
			// continue to next email
		}

		verificationRequest := &schemas.VerificationRequest{
			Token:       verificationToken,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		}

		// use magic link login if that option is on
		if isMagicLinkLoginEnabled {
			user.SignupMethods = constants.AuthRecipeMethodMagicLinkLogin
			verificationRequest.Identifier = constants.VerificationTypeMagicLinkLogin
		} else {
			// use basic authentication if that option is on
			user.SignupMethods = constants.AuthRecipeMethodBasicAuth
			verificationRequest.Identifier = constants.VerificationTypeInviteMember

			isMFAEnforced := p.Config.EnforceMFA
			if isMFAEnforced {
				user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
			}
			verifyEmailURL = appURL + "/setup-password"
		}

		user, err = p.StorageProvider.AddUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add user")
			return nil, nil, err
		}

		_, err = p.StorageProvider.AddVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, nil, err
		}

		// exec it as go routine so that we can reduce the api latency
		go func() {
			_ = p.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeInviteMember, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(p.Config),
				"verification_url": utils.GetInviteVerificationURL(verifyEmailURL, verificationToken, redirectURL),
			})
		}()
	}

	invitedUsers := []*model.User{}
	for _, email := range newEmails {
		user, err := p.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
			return nil, nil, err
		}
		invitedUsers = append(invitedUsers, &model.User{
			Email: user.Email,
			ID:    user.ID,
		})
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminInviteSentEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeUser,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.InviteMembersResponse{
		Message: fmt.Sprintf("%d user(s) invited successfully.", len(newEmails)),
		Users:   invitedUsers,
	}, nil, nil
}
