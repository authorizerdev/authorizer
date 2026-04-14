package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeletePolicy is the method to delete an authorization policy.
// Permissions: authorizer:admin
func (g *graphqlProvider) DeletePolicy(ctx context.Context, id string) (*model.Response, error) {
	log := g.Log.With().Str("func", "DeletePolicy").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("policy ID is required")
	}

	// Delete targets first
	err = g.StorageProvider.DeletePolicyTargetsByPolicyID(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete policy targets")
		return nil, err
	}

	err = g.StorageProvider.DeletePolicy(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete policy")
		return nil, err
	}

	go g.AuthorizationProvider.InvalidateCache(ctx, "authz:")

	return &model.Response{
		Message: "Policy deleted successfully",
	}, nil
}
