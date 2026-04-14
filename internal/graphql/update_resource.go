package graphql

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UpdateResource is the method to update an existing authorization resource.
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdateResource(ctx context.Context, params *model.UpdateResourceInput) (*model.AuthzResource, error) {
	log := g.Log.With().Str("func", "UpdateResource").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if strings.TrimSpace(params.ID) == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	resource, err := g.StorageProvider.GetResourceByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get resource by ID")
		return nil, err
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			return nil, fmt.Errorf("resource name cannot be empty")
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("invalid name: must be 100 characters or fewer")
		}
		for _, r := range name {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
				return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
			}
		}
		resource.Name = name
	}
	if params.Description != nil {
		resource.Description = *params.Description
	}

	resource, err = g.StorageProvider.UpdateResource(ctx, resource)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update resource")
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return resource.AsAPIResource(), nil
}
