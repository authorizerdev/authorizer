package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// ResendVerifyEmailResolver is a resolver for resend verify email mutation
func ResendVerifyEmailResolver(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}
	params.Email = strings.ToLower(params.Email)

	if !validators.IsValidEmail(params.Email) {
		log.Debug("Invalid email: ", params.Email)
		return res, fmt.Errorf("invalid email")
	}

	if !validators.IsValidVerificationIdentifier(params.Identifier) {
		log.Debug("Invalid verification identifier: ", params.Identifier)
		return res, fmt.Errorf("invalid identifier")
	}

	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		return res, fmt.Errorf("invalid user")
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, params.Email, params.Identifier)
	if err != nil {
		log.Debug("Failed to get verification request: ", err)
		return res, fmt.Errorf(`verification request not found`)
	}

	// delete current verification and create new one
	err = db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
	if err != nil {
		log.Debug("Failed to delete verification request: ", err)
	}

	hostname := parsers.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug("Failed to generate nonce: ", err)
		return res, err
	}

	verificationToken, err := token.CreateVerificationToken(params.Email, params.Identifier, hostname, nonceHash, verificationRequest.RedirectURI)
	if err != nil {
		log.Debug("Failed to create verification token: ", err)
	}
	_, err = db.Provider.AddVerificationRequest(ctx, &models.VerificationRequest{
		Token:       verificationToken,
		Identifier:  params.Identifier,
		ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
		Email:       params.Email,
		Nonce:       nonceHash,
		RedirectURI: verificationRequest.RedirectURI,
	})
	if err != nil {
		log.Debug("Failed to add verification request: ", err)
	}

	// exec it as go routine so that we can reduce the api latency
	go email.SendEmail([]string{params.Email}, params.Identifier, map[string]interface{}{
		"user":             user.ToMap(),
		"organization":     utils.GetOrganization(),
		"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, verificationRequest.RedirectURI),
	})

	res = &model.Response{
		Message: `Verification email has been sent. Please check your inbox`,
	}

	return res, nil
}
