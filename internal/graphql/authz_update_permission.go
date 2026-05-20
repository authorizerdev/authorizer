package graphql

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AuthzUpdatePermission is the method to update an existing authorization permission.
// Permissions: authorizer:admin
func (g *graphqlProvider) AuthzUpdatePermission(ctx context.Context, params *model.UpdatePermissionInput) (*model.AuthzPermission, error) {
	log := g.Log.With().Str("func", "AuthzUpdatePermission").Logger()
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

	// Build a copy of the permission with the requested field changes applied.
	// Persistence is deferred until AFTER the link-rebuild loops succeed so that
	// a link-attach failure does not leave the row with new fields but old links.
	newPermission := *permission
	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			return nil, fmt.Errorf("permission name cannot be empty")
		}
		if len(name) > constants.MaxAuthzIdentifierLength {
			return nil, fmt.Errorf("invalid name: must be %d characters or fewer", constants.MaxAuthzIdentifierLength)
		}
		for _, r := range name {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
				return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
			}
		}
		newPermission.Name = name
	}
	if params.Description != nil {
		newPermission.Description = *params.Description
	}
	if params.DecisionStrategy != nil {
		ds := *params.DecisionStrategy
		if ds != constants.DecisionStrategyAffirmative && ds != constants.DecisionStrategyUnanimous {
			return nil, fmt.Errorf("invalid decision strategy: must be '%s' or '%s'",
				constants.DecisionStrategyAffirmative, constants.DecisionStrategyUnanimous)
		}
		newPermission.DecisionStrategy = ds
	}

	var apiScopes []*model.AuthzScope
	if params.ScopeIds != nil {
		if len(params.ScopeIds) == 0 {
			return nil, fmt.Errorf("at least one scope_id is required")
		}
		apiScopes = make([]*model.AuthzScope, 0, len(params.ScopeIds))
		for _, scopeID := range params.ScopeIds {
			scope, err := g.StorageProvider.GetScopeByID(ctx, scopeID)
			if err != nil {
				log.Debug().Err(err).Str("scope_id", scopeID).Msg("Failed to get scope by ID")
				return nil, fmt.Errorf("scope not found: %s", scopeID)
			}
			apiScopes = append(apiScopes, scope.AsAPIScope())
		}
	}

	var apiPolicies []*model.AuthzPolicy
	if params.PolicyIds != nil {
		if len(params.PolicyIds) == 0 {
			return nil, fmt.Errorf("at least one policy_id is required")
		}
		apiPolicies = make([]*model.AuthzPolicy, 0, len(params.PolicyIds))
		for _, policyID := range params.PolicyIds {
			policy, err := g.StorageProvider.GetPolicyByID(ctx, policyID)
			if err != nil {
				log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to get policy by ID")
				return nil, fmt.Errorf("policy not found: %s", policyID)
			}
			targets, err := g.StorageProvider.GetPolicyTargets(ctx, policyID)
			if err != nil {
				log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to get policy targets")
				return nil, err
			}
			apiPolicies = append(apiPolicies, policy.AsAPIPolicy(targets))
		}
	}

	oldScopeLinks, err := g.StorageProvider.GetPermissionScopes(ctx, permission.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get existing permission scopes")
		return nil, err
	}
	oldPolicyLinks, err := g.StorageProvider.GetPermissionPolicies(ctx, permission.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get existing permission policies")
		return nil, err
	}

	// Replace scopes if provided. Delete-then-add ordering avoids accumulating
	// duplicates. On failure, restore the previous link sets and bail out
	// WITHOUT persisting the field changes (newPermission has not been written
	// yet), so the on-disk permission row remains untouched.
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
				g.rollbackPermissionLinks(permission.ID, oldScopeLinks, oldPolicyLinks)
				g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")
				return nil, err
			}
		}
	}

	// Replace policies if provided. Same delete-then-add semantics as scopes.
	if params.PolicyIds != nil {
		err = g.StorageProvider.DeletePermissionPoliciesByPermissionID(ctx, permission.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete existing permission policies")
			g.rollbackPermissionLinks(permission.ID, oldScopeLinks, oldPolicyLinks)
			g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")
			return nil, err
		}
		for _, policyID := range params.PolicyIds {
			_, err := g.StorageProvider.AddPermissionPolicy(ctx, &schemas.PermissionPolicy{
				PermissionID: permission.ID,
				PolicyID:     policyID,
			})
			if err != nil {
				log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to add permission policy")
				g.rollbackPermissionLinks(permission.ID, oldScopeLinks, oldPolicyLinks)
				g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")
				return nil, err
			}
		}
	}

	// Persist the field changes only AFTER both link-rebuild loops have
	// succeeded. If this fails, undo the link replacements so the permission
	// row + its links remain consistent (old fields, old links).
	updated, err := g.StorageProvider.UpdatePermission(ctx, &newPermission)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update permission")
		g.rollbackPermissionLinks(permission.ID, oldScopeLinks, oldPolicyLinks)
		g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")
		return nil, err
	}
	permission = updated

	// Resolve the full permission for the response
	resource, err := g.StorageProvider.GetResourceByID(ctx, permission.ResourceID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get resource")
		return nil, err
	}

	if params.ScopeIds == nil {
		apiScopes, err = g.resolvePermissionScopes(ctx, permission.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to resolve permission scopes")
			return nil, err
		}
	}

	if params.PolicyIds == nil {
		apiPolicies, err = g.resolvePermissionPolicies(ctx, permission.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to resolve permission policies")
			return nil, err
		}
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminAuthzPermissionUpdatedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeAuthzPermission,
		ResourceID:   permission.ID,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})

	return permission.AsAPIPermission(resource.AsAPIResource(), apiScopes, apiPolicies), nil
}

func (g *graphqlProvider) rollbackPermissionLinks(permissionID string, scopes []*schemas.PermissionScope, policies []*schemas.PermissionPolicy) {
	log := g.Log.With().Str("func", "rollbackPermissionLinks").Logger()
	if err := g.StorageProvider.DeletePermissionScopesByPermissionID(context.Background(), permissionID); err != nil {
		log.Error().Err(err).Str("permission_id", permissionID).Msg("failed to delete permission scopes during rollback")
	}
	for _, scope := range scopes {
		if _, err := g.StorageProvider.AddPermissionScope(context.Background(), &schemas.PermissionScope{
			PermissionID: permissionID,
			ScopeID:      scope.ScopeID,
		}); err != nil {
			log.Error().Err(err).Str("permission_id", permissionID).Str("scope_id", scope.ScopeID).Msg("failed to restore permission scope during rollback")
		}
	}
	if err := g.StorageProvider.DeletePermissionPoliciesByPermissionID(context.Background(), permissionID); err != nil {
		log.Error().Err(err).Str("permission_id", permissionID).Msg("failed to delete permission policies during rollback")
	}
	for _, policy := range policies {
		if _, err := g.StorageProvider.AddPermissionPolicy(context.Background(), &schemas.PermissionPolicy{
			PermissionID: permissionID,
			PolicyID:     policy.PolicyID,
		}); err != nil {
			log.Error().Err(err).Str("permission_id", permissionID).Str("policy_id", policy.PolicyID).Msg("failed to restore permission policy during rollback")
		}
	}
}
