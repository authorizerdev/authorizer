package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AddScope is the method to create a new authorization scope.
// Permissions: authorizer:admin
func (g *graphqlProvider) AddScope(ctx context.Context, params *model.AddScopeInput) (*model.AuthzScope, error) {
	log := g.Log.With().Str("func", "AddScope").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, fmt.Errorf("scope name is required")
	}

	description := ""
	if params.Description != nil {
		description = *params.Description
	}

	scope, err := g.StorageProvider.AddScope(ctx, &schemas.Scope{
		Name:        name,
		Description: description,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add scope")
		return nil, err
	}

	go g.AuthorizationProvider.InvalidateCache(ctx, "authz:")

	return scope.AsAPIScope(), nil
}
