package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AddPermission is the method to create a new authorization permission
// binding a resource to scopes and policies.
// Permissions: authorizer:admin
func (g *graphqlProvider) AddPermission(ctx context.Context, params *model.AddPermissionInput) (*model.AuthzPermission, error) {
	log := g.Log.With().Str("func", "AddPermission").Logger()
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
		return nil, fmt.Errorf("permission name is required")
	}

	if strings.TrimSpace(params.ResourceID) == "" {
		return nil, fmt.Errorf("resource_id is required")
	}

	if len(params.ScopeIds) == 0 {
		return nil, fmt.Errorf("at least one scope_id is required")
	}

	if len(params.PolicyIds) == 0 {
		return nil, fmt.Errorf("at least one policy_id is required")
	}

	description := ""
	if params.Description != nil {
		description = *params.Description
	}

	decisionStrategy := "affirmative"
	if params.DecisionStrategy != nil {
		decisionStrategy = *params.DecisionStrategy
	}

	// Verify resource exists
	resource, err := g.StorageProvider.GetResourceByID(ctx, params.ResourceID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get resource by ID")
		return nil, fmt.Errorf("resource not found: %s", params.ResourceID)
	}

	permission, err := g.StorageProvider.AddPermission(ctx, &schemas.Permission{
		Name:             name,
		Description:      description,
		ResourceID:       params.ResourceID,
		DecisionStrategy: decisionStrategy,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add permission")
		return nil, err
	}

	// Add permission scopes
	apiScopes := make([]*model.AuthzScope, 0, len(params.ScopeIds))
	for _, scopeID := range params.ScopeIds {
		_, err := g.StorageProvider.AddPermissionScope(ctx, &schemas.PermissionScope{
			PermissionID: permission.ID,
			ScopeID:      scopeID,
		})
		if err != nil {
			log.Debug().Err(err).Str("scope_id", scopeID).Msg("Failed to add permission scope")
			return nil, err
		}
		scope, err := g.StorageProvider.GetScopeByID(ctx, scopeID)
		if err != nil {
			log.Debug().Err(err).Str("scope_id", scopeID).Msg("Failed to get scope by ID")
			return nil, err
		}
		apiScopes = append(apiScopes, scope.AsAPIScope())
	}

	// Add permission policies
	apiPolicies := make([]*model.AuthzPolicy, 0, len(params.PolicyIds))
	for _, policyID := range params.PolicyIds {
		_, err := g.StorageProvider.AddPermissionPolicy(ctx, &schemas.PermissionPolicy{
			PermissionID: permission.ID,
			PolicyID:     policyID,
		})
		if err != nil {
			log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to add permission policy")
			return nil, err
		}
		policy, err := g.StorageProvider.GetPolicyByID(ctx, policyID)
		if err != nil {
			log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to get policy by ID")
			return nil, err
		}
		targets, err := g.StorageProvider.GetPolicyTargets(ctx, policyID)
		if err != nil {
			log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to get policy targets")
			return nil, err
		}
		apiPolicies = append(apiPolicies, policy.AsAPIPolicy(targets))
	}

	go g.AuthorizationProvider.InvalidateCache(ctx, "authz:")

	return permission.AsAPIPermission(resource.AsAPIResource(), apiScopes, apiPolicies), nil
}
