package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// EnableAccess is the method to enable access for a user.
// Permissions: authorizer:admin
func (s *service) EnableAccess(ctx context.Context, params *model.UpdateAccessInput) (*model.Response, error) {
	log := s.Log.With().Str("func", "EnableAccess").Logger()

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if params.UserID == "" {
		return nil, fmt.Errorf("user ID is missing")
	}

	log = log.With().Str("user_id", params.UserID).Logger()

	user, err := s.StorageProvider.GetUserByID(ctx, params.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by ID")
		return nil, err
	}

	user.RevokedTimestamp = nil

	user, err = s.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, err
	}
	go s.EventsProvider.RegisterEvent(ctx, constants.UserAccessEnabledWebhookEvent, "", user)

	return &model.Response{
		Message: `user access enabled successfully`,
	}, nil
}
