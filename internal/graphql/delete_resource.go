package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeleteResource is the method to delete an authorization resource.
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteResource(ctx context.Context, id string) (*model.Response, error) {
	log := g.Log.With().Str("func", "DeleteResource").Logger()
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
		return nil, fmt.Errorf("resource ID is required")
	}

	err = g.StorageProvider.DeleteResource(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete resource")
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return &model.Response{
		Message: "Resource deleted successfully",
	}, nil
}
