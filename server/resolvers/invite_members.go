package resolvers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	emailservice "github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// InviteMembersResolver resolver to invite members
func InviteMembersResolver(ctx context.Context, params model.InviteMemberInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return nil, errors.New("unauthorized")
	}

	// this feature is only allowed if email server is configured
	if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification) {
		log.Debug("Email server is not configured.")
		return nil, errors.New("email sending is disabled")
	}

	if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication) && envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin) {
		log.Debug("Basic authentication and Magic link login is disabled.")
		return nil, errors.New("either basic authentication or magic link login is required")
	}

	// filter valid emails
	emails := []string{}
	for _, email := range params.Emails {
		if utils.IsValidEmail(email) {
			emails = append(emails, email)
		}
	}

	if len(emails) == 0 {
		log.Debug("No valid email addresses.")
		return nil, errors.New("no valid emails found")
	}

	// TODO: optimise to use like query instead of looping through emails and getting user individually
	// for each emails check if emails exists in db
	newEmails := []string{}
	for _, email := range emails {
		_, err := db.Provider.GetUserByEmail(email)
		if err != nil {
			log.Info("User with this email not found, so inviting...")
			newEmails = append(newEmails, email)
		} else {
			log.Info("User with this email already exists, so not inviting...")
		}
	}

	if len(newEmails) == 0 {
		log.Debug("No new emails found.")
		return nil, errors.New("all emails already exist")
	}

	// invite new emails
	for _, email := range newEmails {

		user := models.User{
			Email: email,
			Roles: strings.Join(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles), ","),
		}
		hostname := utils.GetHost(gc)
		verifyEmailURL := hostname + "/verify_email"
		appURL := utils.GetAppURL(gc)

		redirectURL := appURL
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
		}

		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			return nil, err
		}

		verificationToken, err := token.CreateVerificationToken(email, constants.VerificationTypeForgotPassword, hostname, nonceHash, redirectURL)
		if err != nil {
			log.Debug("Failed to create verification token.", err)
		}

		verificationRequest := models.VerificationRequest{
			Token:       verificationToken,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		}

		// use magic link login if that option is on
		if !envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin) {
			user.SignupMethods = constants.SignupMethodMagicLinkLogin
			verificationRequest.Identifier = constants.VerificationTypeMagicLinkLogin
		} else {
			// use basic authentication if that option is on
			user.SignupMethods = constants.SignupMethodBasicAuth
			verificationRequest.Identifier = constants.VerificationTypeForgotPassword

			verifyEmailURL = appURL + "/setup-password"

		}

		user, err = db.Provider.AddUser(user)
		if err != nil {
			log.Debug("Error adding user: %s, err: %v", email, err)
			return nil, err
		}

		_, err = db.Provider.AddVerificationRequest(verificationRequest)
		if err != nil {
			log.Debug("Error adding verification request: %s, err: %v", email, err)
			return nil, err
		}

		go emailservice.InviteEmail(email, verificationToken, verifyEmailURL, redirectURL)
	}

	return &model.Response{
		Message: fmt.Sprintf("%d user(s) invited successfully.", len(newEmails)),
	}, nil
}
