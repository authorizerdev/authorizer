package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeletePermission is the method to delete an authorization permission.
// Permissions: authorizer:admin
func (g *graphqlProvider) DeletePermission(ctx context.Context, id string) (*model.Response, error) {
	log := g.Log.With().Str("func", "DeletePermission").Logger()
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
		return nil, fmt.Errorf("permission ID is required")
	}

	// Clean up join tables first
	err = g.StorageProvider.DeletePermissionScopesByPermissionID(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete permission scopes")
		return nil, err
	}

	err = g.StorageProvider.DeletePermissionPoliciesByPermissionID(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete permission policies")
		return nil, err
	}

	err = g.StorageProvider.DeletePermission(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete permission")
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return &model.Response{
		Message: "Permission deleted successfully",
	}, nil
}
