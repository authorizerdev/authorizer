package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Profile returns the authenticated user. Requires a valid session cookie or
// access-token bearer. Transport-agnostic port of graphqlProvider.Profile.
//
// Permissions: authenticated user.
func (p *provider) Profile(ctx context.Context, meta RequestMetadata) (*model.User, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Profile").Logger()

	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, nil, Unauthenticated("unauthorized")
	}
	if tokenData == nil || tokenData.UserID == "" {
		return nil, nil, Unauthenticated("unauthorized")
	}
	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Str("user_id", tokenData.UserID).Msg("Failed to get user by id")
		return nil, nil, err
	}
	return user.AsAPIUser(), nil, nil
}
