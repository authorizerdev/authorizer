package service

import (
	"context"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
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

	hostname := meta.HostURL
	// The redirect the new link lands on. Reuse the pending request's when
	// there is one, so a resend behaves exactly like the original mail.
	redirectURI := hostname + "/app"

	verificationRequest, err := p.StorageProvider.GetVerificationRequestByEmail(ctx, params.Email, params.Identifier)
	switch {
	case err == nil && verificationRequest != nil:
		redirectURI = verificationRequest.RedirectURI
		// delete current verification and create new one
		if delErr := p.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest); delErr != nil {
			log.Debug().Err(delErr).Msg("Failed to delete verification request")
		}
	default:
		// No pending request. This used to silently drop, which made the
		// endpoint useless in exactly the situation it exists for: verification
		// requests expire after 30 minutes, so a user who came back later had no
		// way to ask for a new link at all — and no way to verify their address,
		// since the link is the only self-service proof of mailbox control.
		//
		// Mint a fresh one instead, but only when there is genuinely something
		// to verify. Resending to an already-verified address would let anyone
		// who knows it use this endpoint as a mailer.
		if !p.resendNeedsFreshVerification(user, params.Identifier) {
			log.Debug().Str("reason", "nothing_to_verify").Msg("resend verify email silently dropped")
			return genericResponse, nil, nil
		}
		log.Debug().Msg("no pending verification request; minting a fresh one")
	}

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
	}, redirectURI, params.Identifier)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create verification token")
	}
	_, err = p.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
		Token:       verificationToken,
		Identifier:  params.Identifier,
		ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
		Email:       params.Email,
		Nonce:       nonceHash,
		RedirectURI: redirectURI,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add verification request")
	}

	// exec it as go routine so that we can reduce the api latency
	asyncutil.Go(p.Log, func() {
		_ = p.EmailProvider.SendEmail([]string{params.Email}, params.Identifier, map[string]any{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(p.Config),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURI),
		})
	})
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

// resendNeedsFreshVerification reports whether minting a brand-new verification
// request for this user is warranted when none is pending.
//
// Only the email-verification family qualifies, and only when the address is
// actually still unverified. Without this, the endpoint would happily mail a
// fresh link to any address a caller names — including already-verified ones —
// turning it into an open mailer for anyone who knows a registered address.
//
// Magic-link login is deliberately excluded: it has its own entry point
// (MagicLinkLogin) which applies its own policy, and a "resend" that mints a
// login link on demand is a login endpoint wearing a different name.
func (p *provider) resendNeedsFreshVerification(user *schemas.User, identifier string) bool {
	if user == nil {
		return false
	}
	switch identifier {
	case constants.VerificationTypeBasicAuthSignup,
		constants.VerificationTypeUpdateEmail,
		constants.VerificationTypeInviteMember:
		return user.EmailVerifiedAt == nil
	default:
		return false
	}
}
