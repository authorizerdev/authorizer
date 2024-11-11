package resolvers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	emailservice "github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/models/db"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// InviteMembersResolver resolver to invite members
func InviteMembersResolver(ctx context.Context, params model.InviteMemberInput) (*model.InviteMembersResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return nil, errors.New("unauthorized")
	}

	// this feature is only allowed if email server is configured
	EnvKeyIsEmailServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
	if err != nil {
		log.Debug("Error getting email verification disabled: ", err)
		EnvKeyIsEmailServiceEnabled = false
	}

	if !EnvKeyIsEmailServiceEnabled {
		log.Debug("Email server is not configured")
		return nil, errors.New("email sending is disabled")
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Failed to get is basic auth disabled")
		return nil, err
	}
	isMagicLinkLoginDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin)
	if err != nil {
		log.Debug("Failed to get is magic link login disabled")
		return nil, err
	}
	if isBasicAuthDisabled && isMagicLinkLoginDisabled {
		log.Debug("Basic authentication and Magic link login is disabled.")
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
		log.Debug("No valid email addresses")
		return nil, errors.New("no valid emails found")
	}

	// TODO: optimise to use like query instead of looping through emails and getting user individually
	// for each emails check if emails exists in db
	newEmails := []string{}
	for _, email := range emails {
		_, err := db.Provider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debugf("User with %s email not found, so inviting user", email)
			newEmails = append(newEmails, email)
		} else {
			log.Debugf("User with %s email already exists, so not inviting user", email)
		}
	}

	if len(newEmails) == 0 {
		log.Debug("No new emails found.")
		return nil, errors.New("all emails already exist")
	}

	// invite new emails
	for _, email := range newEmails {

		defaultRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
		defaultRoles := []string{}
		if err != nil {
			log.Debug("Error getting default roles: ", err)
			defaultRolesString = ""
		} else {
			defaultRoles = strings.Split(defaultRolesString, ",")
		}

		user := &models.User{
			Email: refs.NewStringRef(email),
			Roles: strings.Join(defaultRoles, ","),
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

		verificationToken, err := token.CreateVerificationToken(email, constants.VerificationTypeInviteMember, hostname, nonceHash, redirectURL)
		if err != nil {
			log.Debug("Failed to create verification token: ", err)
		}

		verificationRequest := &models.VerificationRequest{
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

			isMFAEnforced, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyEnforceMultiFactorAuthentication)
			if err != nil {
				log.Debug("MFA service not enabled: ", err)
				isMFAEnforced = false
			}

			if isMFAEnforced {
				user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
			}
			verifyEmailURL = appURL + "/setup-password"

		}

		user, err = db.Provider.AddUser(ctx, user)
		if err != nil {
			log.Debugf("Error adding user: %s, err: %v", email, err)
			return nil, err
		}

		_, err = db.Provider.AddVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debugf("Error adding verification request: %s, err: %v", email, err)
			return nil, err
		}

		// exec it as go routine so that we can reduce the api latency
		go emailservice.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeInviteMember, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(),
			"verification_url": utils.GetInviteVerificationURL(verifyEmailURL, verificationToken, redirectURL),
		})
	}

	InvitedUsers := []*model.User{}

	for _, email := range newEmails {
		user, err := db.Provider.GetUserByEmail(ctx, email)

		if err != nil {
			log.Debugf("err: %s", err.Error())
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
