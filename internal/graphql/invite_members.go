package graphql

import (
	"context"
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

// InviteMembers is the method to invite members to the organization.
// Permissions: authorizer:admin
func (g *graphqlProvider) InviteMembers(ctx context.Context, params *model.InviteMemberInput) (*model.InviteMembersResponse, error) {
	log := g.Log.With().Str("func", "InviteMembers").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	// this feature is only allowed if email server is configured
	if !g.Config.IsEmailServiceEnabled {
		log.Debug().Msg("Email sending is disabled")
		return nil, errors.New("email sending is disabled")
	}

	isBasicAuthDisabled := g.Config.DisableBasicAuthentication
	isMagicLinkLoginDisabled := g.Config.DisableMagicLinkLogin
	if isBasicAuthDisabled && isMagicLinkLoginDisabled {
		log.Debug().Msg("Either basic authentication or magic link login is required")
		return nil, errors.New("either basic authentication or magic link login is required")
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
		return nil, errors.New("no valid emails found")
	}

	// TODO: optimise to use like query instead of looping through emails and getting user individually
	// for each emails check if emails exists in db
	newEmails := []string{}
	for _, email := range emails {
		_, err := g.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Msgf("User with %s email does not exist", email)
			newEmails = append(newEmails, email)
		} else {
			log.Debug().Msgf("User with %s email already exists", email)
		}
	}

	if len(newEmails) == 0 {
		log.Debug().Msg("All emails already exist")
		return nil, errors.New("all emails already exist")
	}

	// invite new emails
	for _, email := range newEmails {
		user := &schemas.User{
			Email: refs.NewStringRef(email),
			Roles: strings.Join(g.Config.DefaultRoles, ","),
		}
		hostname := parsers.GetHost(gc)
		verifyEmailURL := hostname + "/verify_email"
		appURL := parsers.GetAppURL(gc)

		redirectURL := appURL
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
		}

		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			return nil, err
		}
		// email, constants.VerificationTypeInviteMember, hostname, nonceHash, redirectURL

		verificationToken, err := g.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
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
		if !isMagicLinkLoginDisabled {
			user.SignupMethods = constants.AuthRecipeMethodMagicLinkLogin
			verificationRequest.Identifier = constants.VerificationTypeMagicLinkLogin
		} else {
			// use basic authentication if that option is on
			user.SignupMethods = constants.AuthRecipeMethodBasicAuth
			verificationRequest.Identifier = constants.VerificationTypeInviteMember

			isMFAEnforced := g.Config.EnforceMFA
			if isMFAEnforced {
				user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
			}
			verifyEmailURL = appURL + "/setup-password"
		}

		user, err = g.StorageProvider.AddUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add user")
			return nil, err
		}

		_, err = g.StorageProvider.AddVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, err
		}

		// exec it as go routine so that we can reduce the api latency
		go g.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeInviteMember, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(g.Config),
			"verification_url": utils.GetInviteVerificationURL(verifyEmailURL, verificationToken, redirectURL),
		})
	}

	InvitedUsers := []*model.User{}

	for _, email := range newEmails {
		user, err := g.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
			return nil, err
		}
		InvitedUsers = append(InvitedUsers, &model.User{
			Email: user.Email,
			ID:    user.ID,
		})
	}

	return &model.InviteMembersResponse{
		Message: fmt.Sprintf("%d user(s) invited successfully.", len(newEmails)),
		Users:   InvitedUsers,
	}, nil
}
