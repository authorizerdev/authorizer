// internal/service/skip_mfa_setup.go
package service

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// SkipMFASetup records that the authenticated caller explicitly declined the
// optional MFA setup prompt shown at login. Never allowed when MFA is
// org-enforced — that path never offers a skip in the first place
// (resolveMFAGate never returns mfaGateOfferSetup when EnforceMFA is true),
// but this is re-checked here server-side so a client can never forge the
// request to bypass enforcement.
//
// Permissions: authenticated user (bearer token or session cookie).
func (p *provider) SkipMFASetup(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "SkipMFASetup").Logger()

	// Authentication is checked before EnforceMFA so the response code never
	// leaks org-wide MFA enforcement to a caller with no valid token/session
	// (an unauthenticated caller always gets Unauthenticated, regardless of
	// EnforceMFA). EnforceMFA is still re-checked below, before any state
	// mutation, so HasSkippedMFASetupAt is never set while it is true.
	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, nil, Unauthenticated("unauthorized")
	}

	if p.Config.EnforceMFA {
		log.Debug().Msg("Cannot skip MFA setup as it is enforced")
		return nil, nil, FailedPrecondition("cannot skip multi factor authentication setup as it is enforced by organization")
	}

	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, nil, err
	}
	now := time.Now().Unix()
	user.HasSkippedMFASetupAt = &now
	if _, err := p.StorageProvider.UpdateUser(ctx, user); err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}
	return &model.Response{Message: "MFA setup skipped"}, nil, nil
}
