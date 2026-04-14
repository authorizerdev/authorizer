package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Scopes is the method to list authorization scopes with pagination.
// Permissions: authorizer:admin
func (g *graphqlProvider) Scopes(ctx context.Context, params *model.PaginatedRequest) (*model.AuthzScopes, error) {
	log := g.Log.With().Str("func", "Scopes").Logger()
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
	scopes, pagination, err := g.StorageProvider.ListScopes(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list scopes")
		return nil, err
	}

	res := make([]*model.AuthzScope, len(scopes))
	for i, s := range scopes {
		res[i] = s.AsAPIScope()
	}

	return &model.AuthzScopes{
		Pagination: pagination,
		Scopes:     res,
	}, nil
}
