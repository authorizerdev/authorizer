package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Users is the method to get list of users
// Permission: authorizer:admin
func (g *graphqlProvider) Users(ctx context.Context, params *model.PaginatedRequest) (*model.Users, error) {
	log := g.Log.With().Str("func", "Users").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	res, pagination, err := g.StorageProvider.ListUsers(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListUsers")
		return nil, err
	}
	resItems := make([]*model.User, len(res))
	for i, user := range res {
		resItems[i] = user.AsAPIUser()
	}
	return &model.Users{
		Pagination: pagination,
		Users:      resItems,
	}, nil
}
