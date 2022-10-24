package resolvers

import (
	"context"
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

// ForgotPasswordResolver is a resolver for forgot password mutation
func ForgotPasswordResolver(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	if isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	params.Email = strings.ToLower(params.Email)

	if !validators.IsValidEmail(params.Email) {
		log.Debug("Invalid email address: ", params.Email)
		return res, fmt.Errorf("invalid email")
	}

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})
	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("User not found: ", err)
		return res, fmt.Errorf(`user with this email not found`)
	}

	hostname := parsers.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug("Failed to generate nonce: ", err)
		return res, err
	}

	redirectURI := ""
	// give higher preference to params redirect uri
	if strings.TrimSpace(refs.StringValue(params.RedirectURI)) != "" {
		fmt.Println("=> redirect uri from here1", redirectURI)
		redirectURI = refs.StringValue(params.RedirectURI)
	} else {
		redirectURI, err = memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyResetPasswordURL)
		fmt.Println("=> redirect uri from here2", redirectURI)
		if err != nil {
			log.Debug("ResetPasswordURL not found using default app url: ", err)
			redirectURI = hostname + "/app/reset-password"
			fmt.Println("=> redirect uri from here3", redirectURI)
			memorystore.Provider.UpdateEnvVariable(constants.EnvKeyResetPasswordURL, redirectURI)
		}
	}

	fmt.Println("=> redirect uri", redirectURI)

	verificationToken, err := token.CreateVerificationToken(params.Email, constants.VerificationTypeForgotPassword, hostname, nonceHash, redirectURI)
	if err != nil {
		log.Debug("Failed to create verification token", err)
		return res, err
	}
	_, err = db.Provider.AddVerificationRequest(ctx, models.VerificationRequest{
		Token:       verificationToken,
		Identifier:  constants.VerificationTypeForgotPassword,
		ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
		Email:       params.Email,
		Nonce:       nonceHash,
		RedirectURI: redirectURI,
	})
	if err != nil {
		log.Debug("Failed to add verification request", err)
		return res, err
	}

	// exec it as go routine so that we can reduce the api latency
	go email.SendEmail([]string{params.Email}, constants.VerificationTypeForgotPassword, map[string]interface{}{
		"user":             user.ToMap(),
		"organization":     utils.GetOrganization(),
		"verification_url": utils.GetForgotPasswordURL(verificationToken, redirectURI),
	})

	res = &model.Response{
		Message: `Please check your inbox! We have sent a password reset link.`,
	}

	return res, nil
}
