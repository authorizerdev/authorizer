package service

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// RevokeAccess is the method to revoke access of a user.
// Permission: authorizer:admin
func (s *service) RevokeAccess(ctx context.Context, params *model.UpdateAccessInput) (*model.Response, error) {
	log := s.Log.With().Str("func", "RevokeAccess").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	log = log.With().Str("user_id", params.UserID).Logger()
	user, err := s.StorageProvider.GetUserByID(ctx, params.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, err
	}

	now := time.Now().Unix()
	user.RevokedTimestamp = &now

	user, err = s.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, err
	}

	go func() {
		s.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		s.EventsProvider.RegisterEvent(ctx, constants.UserAccessRevokedWebhookEvent, "", user)
	}()

	return &model.Response{
		Message: `user access revoked successfully`,
	}, nil
}
