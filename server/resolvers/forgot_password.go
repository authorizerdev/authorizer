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
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ForgotPasswordResolver is a resolver for forgot password mutation
func ForgotPasswordResolver(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication) {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		log.Debug("Invalid email address: ", params.Email)
		return res, fmt.Errorf("invalid email")
	}

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})
	_, err = db.Provider.GetUserByEmail(params.Email)
	if err != nil {
		log.Debug("User not found: ", err)
		return res, fmt.Errorf(`user with this email not found`)
	}

	hostname := utils.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug("Failed to generate nonce: ", err)
		return res, err
	}
	redirectURL := utils.GetAppURL(gc) + "/reset-password"
	if params.RedirectURI != nil {
		redirectURL = *params.RedirectURI
	}

	verificationToken, err := token.CreateVerificationToken(params.Email, constants.VerificationTypeForgotPassword, hostname, nonceHash, redirectURL)
	if err != nil {
		log.Debug("Failed to create verification token", err)
		return res, err
	}
	_, err = db.Provider.AddVerificationRequest(models.VerificationRequest{
		Token:       verificationToken,
		Identifier:  constants.VerificationTypeForgotPassword,
		ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
		Email:       params.Email,
		Nonce:       nonceHash,
		RedirectURI: redirectURL,
	})
	if err != nil {
		log.Debug("Failed to add verification request", err)
		return res, err
	}

	// exec it as go routin so that we can reduce the api latency
	go email.SendForgotPasswordMail(params.Email, verificationToken, hostname)

	res = &model.Response{
		Message: `Please check your inbox! We have sent a password reset link.`,
	}

	return res, nil
}
