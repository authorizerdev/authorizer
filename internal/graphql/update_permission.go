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

// UpdatePermission is the method to update an existing authorization permission.
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdatePermission(ctx context.Context, params *model.UpdatePermissionInput) (*model.AuthzPermission, error) {
	log := g.Log.With().Str("func", "UpdatePermission").Logger()
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
		return nil, fmt.Errorf("permission ID is required")
	}

	permission, err := g.StorageProvider.GetPermissionByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get permission by ID")
		return nil, err
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			return nil, fmt.Errorf("permission name cannot be empty")
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("invalid name: must be 100 characters or fewer")
		}
		for _, r := range name {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
				return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
			}
		}
		permission.Name = name
	}
	if params.Description != nil {
		permission.Description = *params.Description
	}
	if params.DecisionStrategy != nil {
		ds := *params.DecisionStrategy
		if ds != "affirmative" && ds != "unanimous" {
			return nil, fmt.Errorf("invalid decision strategy: must be 'affirmative' or 'unanimous'")
		}
		permission.DecisionStrategy = ds
	}

	permission, err = g.StorageProvider.UpdatePermission(ctx, permission)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update permission")
		return nil, err
	}

	// Replace scopes if provided
	if params.ScopeIds != nil {
		err = g.StorageProvider.DeletePermissionScopesByPermissionID(ctx, permission.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete existing permission scopes")
			return nil, err
		}
		for _, scopeID := range params.ScopeIds {
			_, err := g.StorageProvider.AddPermissionScope(ctx, &schemas.PermissionScope{
				PermissionID: permission.ID,
				ScopeID:      scopeID,
			})
			if err != nil {
				log.Debug().Err(err).Str("scope_id", scopeID).Msg("Failed to add permission scope")
				return nil, err
			}
		}
	}

	// Replace policies if provided
	if params.PolicyIds != nil {
		err = g.StorageProvider.DeletePermissionPoliciesByPermissionID(ctx, permission.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete existing permission policies")
			return nil, err
		}
		for _, policyID := range params.PolicyIds {
			_, err := g.StorageProvider.AddPermissionPolicy(ctx, &schemas.PermissionPolicy{
				PermissionID: permission.ID,
				PolicyID:     policyID,
			})
			if err != nil {
				log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to add permission policy")
				return nil, err
			}
		}
	}

	// Resolve the full permission for the response
	resource, err := g.StorageProvider.GetResourceByID(ctx, permission.ResourceID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get resource")
		return nil, err
	}

	apiScopes, err := g.resolvePermissionScopes(ctx, permission.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve permission scopes")
		return nil, err
	}

	apiPolicies, err := g.resolvePermissionPolicies(ctx, permission.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve permission policies")
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return permission.AsAPIPermission(resource.AsAPIResource(), apiScopes, apiPolicies), nil
}
