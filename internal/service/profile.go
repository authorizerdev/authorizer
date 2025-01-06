package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Profile is the method to get the profile of a user.
func (s *service) Profile(ctx context.Context) (*model.User, error) {
	log := s.Log.With().Str("func", "Profile").Logger()

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	tokenData, err := s.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, err
	}
	log = log.With().Str("user_id", tokenData.UserID).Logger()
	user, err := s.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, err
	}

	return user.AsAPIUser(), nil
}
