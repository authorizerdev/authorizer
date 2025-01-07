package graphql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// ResendVerifyEmail is the method to resend verification email.
// Permissions: none
func (g *graphqlProvider) ResendVerifyEmail(ctx context.Context, params *model.ResendVerifyEmailInput) (*model.Response, error) {
	log := g.Log.With().Str("func", "ResendVerifyEmail").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	params.Email = strings.ToLower(params.Email)

	log = log.With().Str("email", params.Email).Str("identifier", params.Identifier).Logger()
	if !validators.IsValidEmail(params.Email) {
		log.Debug().Msg("Invalid email")
		return nil, fmt.Errorf("invalid email")
	}

	if !validators.IsValidVerificationIdentifier(params.Identifier) {
		log.Debug().Msg("Invalid verification identifier")
		return nil, fmt.Errorf("invalid identifier")
	}

	user, err := g.StorageProvider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by email")
		return nil, fmt.Errorf("invalid user")
	}

	verificationRequest, err := g.StorageProvider.GetVerificationRequestByEmail(ctx, params.Email, params.Identifier)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get verification request")
		return nil, fmt.Errorf(`verification request not found`)
	}

	// delete current verification and create new one
	err = g.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete verification request")
	}

	hostname := parsers.GetHost(gc)
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug().Msg("Failed to generate nonce")
		return nil, err
	}
	// params.Email, params.Identifier, hostname, nonceHash, verificationRequest.RedirectURI
	verificationToken, err := g.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonceHash,
		HostName:    hostname,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	}, verificationRequest.RedirectURI, params.Identifier)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create verification token")
	}
	_, err = g.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
		Token:       verificationToken,
		Identifier:  params.Identifier,
		ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
		Email:       params.Email,
		Nonce:       nonceHash,
		RedirectURI: verificationRequest.RedirectURI,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add verification request")
	}

	// exec it as go routine so that we can reduce the api latency
	go g.EmailProvider.SendEmail([]string{params.Email}, params.Identifier, map[string]interface{}{
		"user":             user.ToMap(),
		"organization":     utils.GetOrganization(g.Config),
		"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, verificationRequest.RedirectURI),
	})

	return &model.Response{
		Message: `Verification email has been sent. Please check your inbox`,
	}, nil
}
