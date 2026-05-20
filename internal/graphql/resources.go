package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Resources is the method to list authorization resources with pagination.
// Permissions: authorizer:admin
func (g *graphqlProvider) Resources(ctx context.Context, params *model.PaginatedRequest) (*model.AuthzResources, error) {
	log := g.Log.With().Str("func", "Resources").Logger()
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
	resources, pagination, err := g.StorageProvider.ListResources(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list resources")
		return nil, err
	}

	res := make([]*model.AuthzResource, len(resources))
	for i, r := range resources {
		res[i] = r.AsAPIResource()
	}

	return &model.AuthzResources{
		Pagination: pagination,
		Resources:  res,
	}, nil
}
