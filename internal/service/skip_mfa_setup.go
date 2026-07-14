// internal/service/skip_mfa_setup.go
package service

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// SkipMFASetup completes a token-withheld mfaGateOfferAll offer: it records
// that the caller declined every offered MFA method, then issues the token
// that was withheld at login/signup/oauth-callback time. Permissions: none —
// like VerifyOTP, it completes an in-progress authentication identified by
// the MFA session cookie plus email/phone_number, not a bearer token.
func (p *provider) SkipMFASetup(ctx context.Context, meta RequestMetadata, params *model.SkipMfaSetupRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "SkipMFASetup").Logger()
	side := &ResponseSideEffects{}

	gc := &gin.Context{Request: meta.Request}
	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
	}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument(`email or phone number is required`)
	}

	var user *schemas.User
	if email != "" {
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
	}
	if user == nil || err != nil {
		log.Debug().Err(err).Msg("User not found")
		return nil, nil, NotFound("invalid request")
	}

	// Validate the MFA session before touching any state — same ordering
	// rationale as VerifyOTP: proves the caller actually completed the
	// password/passkey step for THIS user before we act on their behalf.
	if _, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
	}

	if p.Config.EnforceMFA {
		log.Debug().Msg("Cannot skip MFA setup as it is enforced")
		return nil, nil, FailedPrecondition("cannot skip multi factor authentication setup as it is enforced by organization")
	}

	now := time.Now().Unix()
	user.HasSkippedMFASetupAt = &now
	user, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}

	// Known simplification: issueAuthResponse always stamps loginMethod into
	// the audit/webhook trail. The caller may have actually arrived via
	// passkey or OAuth, not password, but issueAuthResponse has no way to
	// recover the original login method from the MFA session today. Out of
	// scope for this task.
	res, err := p.issueAuthResponse(ctx, meta, side, user, constants.AuthRecipeMethodBasicAuth, "MFA setup skipped", params.State, false)
	if err != nil {
		return nil, nil, err
	}
	return res, side, nil
}
