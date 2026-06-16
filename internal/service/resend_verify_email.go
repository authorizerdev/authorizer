package service

import (
	"context"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// ResendVerifyEmail re-issues a pending email-verification link. The response
// is deliberately generic so a caller cannot probe whether an account (or a
// pending verification) exists. Transport-agnostic port of
// graphqlProvider.ResendVerifyEmail.
//
// Permissions: none.
func (p *provider) ResendVerifyEmail(ctx context.Context, meta RequestMetadata, params *model.ResendVerifyEmailRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ResendVerifyEmail").Logger()

	params.Email = strings.ToLower(params.Email)

	log = log.With().Str("email", params.Email).Str("identifier", params.Identifier).Logger()
	if !validators.IsValidEmail(params.Email) {
		log.Debug().Msg("Invalid email")
		return nil, nil, InvalidArgument("invalid email")
	}

	if !validators.IsValidVerificationIdentifier(params.Identifier) {
		log.Debug().Msg("Invalid verification identifier")
		return nil, nil, InvalidArgument("invalid identifier")
	}

	// Do not reveal whether the account or its pending verification exists.
	// Return the same generic response in every code path — including the
	// success path further down — so the user cannot tell from the response
	// alone whether the email matched a real account. The real reason is
	// logged at debug level.
	genericResponse := &model.Response{
		Message: `If a verification is pending for this email, a new link has been sent. Please check your inbox. If you don't receive it within a few minutes, double-check the email address for typos.`,
	}

	user, err := p.StorageProvider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug().Err(err).Str("reason", "user_not_found").Msg("resend verify email silently dropped")
		return genericResponse, nil, nil
	}

	verificationRequest, err := p.StorageProvider.GetVerificationRequestByEmail(ctx, params.Email, params.Identifier)
	if err != nil {
		log.Debug().Err(err).Str("reason", "verification_request_not_found").Msg("resend verify email silently dropped")
		return genericResponse, nil, nil
	}

	// delete current verification and create new one
	err = p.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete verification request")
	}

	hostname := meta.HostURL
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug().Msg("Failed to generate nonce")
		return nil, nil, err
	}
	verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonceHash,
		HostName:    hostname,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	}, verificationRequest.RedirectURI, params.Identifier)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create verification token")
	}
	_, err = p.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
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
	go func() {
		_ = p.EmailProvider.SendEmail([]string{params.Email}, params.Identifier, map[string]any{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(p.Config),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, verificationRequest.RedirectURI),
		})
	}()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditVerifyEmailResentEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   params.Email,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return genericResponse, nil, nil
}
