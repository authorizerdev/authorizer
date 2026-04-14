package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeleteScope is the method to delete an authorization scope.
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteScope(ctx context.Context, id string) (*model.Response, error) {
	log := g.Log.With().Str("func", "DeleteScope").Logger()
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
		return nil, fmt.Errorf("scope ID is required")
	}

	err = g.StorageProvider.DeleteScope(ctx, id)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete scope")
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return &model.Response{
		Message: "Scope deleted successfully",
	}, nil
}
