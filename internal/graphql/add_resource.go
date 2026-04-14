package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AddResource is the method to create a new authorization resource.
// Permissions: authorizer:admin
func (g *graphqlProvider) AddResource(ctx context.Context, params *model.AddResourceInput) (*model.AuthzResource, error) {
	log := g.Log.With().Str("func", "AddResource").Logger()
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
		return nil, fmt.Errorf("resource name is required")
	}

	description := ""
	if params.Description != nil {
		description = *params.Description
	}

	resource, err := g.StorageProvider.AddResource(ctx, &schemas.Resource{
		Name:        name,
		Description: description,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add resource")
		return nil, err
	}

	go g.AuthorizationProvider.InvalidateCache(ctx, "authz:")

	return resource.AsAPIResource(), nil
}
