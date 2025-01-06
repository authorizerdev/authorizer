package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Logout is the method to logout a user.
// Permissions: authenticated:*
func (s *service) Logout(ctx context.Context) (*model.Response, error) {
	log := s.Log.With().Str("func", "Logout").Logger()
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

	sessionKey := tokenData.UserID
	if tokenData.LoginMethod != "" {
		sessionKey = tokenData.LoginMethod + ":" + tokenData.UserID
	}

	s.MemoryStoreProvider.DeleteUserSession(sessionKey, tokenData.Nonce)
	cookie.DeleteSession(gc)

	res := &model.Response{
		Message: "Logged out successfully",
	}

	return res, nil
}
