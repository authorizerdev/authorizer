package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Users is the method to get list of users
// Permission: authorizer:admin
func (s *service) Users(ctx context.Context, params *model.PaginatedInput) (*model.Users, error) {
	log := s.Log.With().Str("func", "Users").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	res, err := s.StorageProvider.ListUsers(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListUsers")
		return nil, err
	}

	return res, nil
}
