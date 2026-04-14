package graphql

import (
	"context"
	"fmt"
	"strings"
	"unicode"

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
	if len(name) > 100 {
		return nil, fmt.Errorf("invalid name: must be 100 characters or fewer")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
		}
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

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return resource.AsAPIResource(), nil
}
