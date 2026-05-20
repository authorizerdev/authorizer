package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Permissions is the method to list authorization permissions with pagination.
// Permissions: authorizer:admin
func (g *graphqlProvider) Permissions(ctx context.Context, params *model.PaginatedRequest) (*model.AuthzPermissions, error) {
	log := g.Log.With().Str("func", "Permissions").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)
	permissions, pagination, err := g.StorageProvider.ListPermissions(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list permissions")
		return nil, err
	}

	res := make([]*model.AuthzPermission, len(permissions))
	for i, p := range permissions {
		resource, err := g.StorageProvider.GetResourceByID(ctx, p.ResourceID)
		if err != nil {
			log.Debug().Err(err).Str("resource_id", p.ResourceID).Msg("Failed to get resource")
			return nil, err
		}

		apiScopes, err := g.resolvePermissionScopes(ctx, p.ID)
		if err != nil {
			log.Debug().Err(err).Str("permission_id", p.ID).Msg("Failed to resolve permission scopes")
			return nil, err
		}

		apiPolicies, err := g.resolvePermissionPolicies(ctx, p.ID)
		if err != nil {
			log.Debug().Err(err).Str("permission_id", p.ID).Msg("Failed to resolve permission policies")
			return nil, err
		}

		res[i] = p.AsAPIPermission(resource.AsAPIResource(), apiScopes, apiPolicies)
	}

	return &model.AuthzPermissions{
		Pagination:  pagination,
		Permissions: res,
	}, nil
}

// resolvePermissionScopes resolves the scopes for a permission.
func (g *graphqlProvider) resolvePermissionScopes(ctx context.Context, permissionID string) ([]*model.AuthzScope, error) {
	permissionScopes, err := g.StorageProvider.GetPermissionScopes(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	apiScopes := make([]*model.AuthzScope, 0, len(permissionScopes))
	for _, ps := range permissionScopes {
		scope, err := g.StorageProvider.GetScopeByID(ctx, ps.ScopeID)
		if err != nil {
			return nil, err
		}
		apiScopes = append(apiScopes, scope.AsAPIScope())
	}
	return apiScopes, nil
}

// resolvePermissionPolicies resolves the policies with their targets for a permission.
func (g *graphqlProvider) resolvePermissionPolicies(ctx context.Context, permissionID string) ([]*model.AuthzPolicy, error) {
	permissionPolicies, err := g.StorageProvider.GetPermissionPolicies(ctx, permissionID)
	if err != nil {
		return nil, err
	}
	apiPolicies := make([]*model.AuthzPolicy, 0, len(permissionPolicies))
	for _, pp := range permissionPolicies {
		policy, err := g.StorageProvider.GetPolicyByID(ctx, pp.PolicyID)
		if err != nil {
			return nil, err
		}
		targets, err := g.StorageProvider.GetPolicyTargets(ctx, policy.ID)
		if err != nil {
			return nil, err
		}
		apiPolicies = append(apiPolicies, policy.AsAPIPolicy(targets))
	}
	return apiPolicies, nil
}
