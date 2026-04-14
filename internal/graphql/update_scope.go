package graphql

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UpdateScope is the method to update an existing authorization scope.
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdateScope(ctx context.Context, params *model.UpdateScopeInput) (*model.AuthzScope, error) {
	log := g.Log.With().Str("func", "UpdateScope").Logger()
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
		return nil, fmt.Errorf("scope ID is required")
	}

	scope, err := g.StorageProvider.GetScopeByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get scope by ID")
		return nil, err
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			return nil, fmt.Errorf("scope name cannot be empty")
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("invalid name: must be 100 characters or fewer")
		}
		for _, r := range name {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
				return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
			}
		}
		scope.Name = name
	}
	if params.Description != nil {
		scope.Description = *params.Description
	}

	scope, err = g.StorageProvider.UpdateScope(ctx, scope)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update scope")
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return scope.AsAPIScope(), nil
}
