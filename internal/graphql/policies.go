package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Policies is the method to list authorization policies with pagination.
// Permissions: authorizer:admin
func (g *graphqlProvider) Policies(ctx context.Context, params *model.PaginatedRequest) (*model.AuthzPolicies, error) {
	log := g.Log.With().Str("func", "Policies").Logger()
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
	policies, pagination, err := g.StorageProvider.ListPolicies(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list policies")
		return nil, err
	}

	res := make([]*model.AuthzPolicy, len(policies))
	for i, p := range policies {
		targets, err := g.StorageProvider.GetPolicyTargets(ctx, p.ID)
		if err != nil {
			log.Debug().Err(err).Str("policy_id", p.ID).Msg("Failed to get policy targets")
			return nil, err
		}
		res[i] = p.AsAPIPolicy(targets)
	}

	return &model.AuthzPolicies{
		Pagination: pagination,
		Policies:   res,
	}, nil
}
