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
	if len(name) > constants.MaxAuthzIdentifierLength {
		return nil, fmt.Errorf("invalid name: must be %d characters or fewer", constants.MaxAuthzIdentifierLength)
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
		}
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

	decisionStrategy := constants.DecisionStrategyAffirmative
	if params.DecisionStrategy != nil {
		decisionStrategy = *params.DecisionStrategy
	}
	if decisionStrategy != constants.DecisionStrategyAffirmative && decisionStrategy != constants.DecisionStrategyUnanimous {
		return nil, fmt.Errorf("invalid decision strategy: must be '%s' or '%s'",
			constants.DecisionStrategyAffirmative, constants.DecisionStrategyUnanimous)
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

	// Attach scopes + policies. The storage layer does not expose transactions
	// across these provider-level calls, so a failure mid-attach would leave
	// the newly created permission row orphaned (present but with partial or
	// no scope/policy links). To keep the system consistent we compensate by
	// deleting the permission when any attach step fails. The delete uses
	// context.Background() so it survives request cancellation (mirrors the
	// pattern already used for InvalidateCache below). If the compensation
	// itself fails, log at ERROR level so operators can manually clean up,
	// but still return the ORIGINAL error — that is the failure operators
	// need to see first.
	apiScopes, apiPolicies, err := g.attachPermissionScopesAndPolicies(ctx, permission.ID, params)
	if err != nil {
		if delErr := g.StorageProvider.DeletePermission(context.Background(), permission.ID); delErr != nil {
			log.Error().
				Err(delErr).
				Str("permission_id", permission.ID).
				Msg("failed to roll back orphaned permission after partial AddPermission failure; manual cleanup required")
		}
		return nil, err
	}

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminAuthzPermissionCreatedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeAuthzPermission,
		ResourceID:   permission.ID,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})

	return permission.AsAPIPermission(resource.AsAPIResource(), apiScopes, apiPolicies), nil
}

// attachPermissionScopesAndPolicies creates PermissionScope and PermissionPolicy
// link rows for a newly added permission and returns the API-shape scope and
// policy slices used to build the GraphQL response. It returns the first error
// encountered so the caller can roll back the permission.
func (g *graphqlProvider) attachPermissionScopesAndPolicies(
	ctx context.Context,
	permissionID string,
	params *model.AddPermissionInput,
) ([]*model.AuthzScope, []*model.AuthzPolicy, error) {
	log := g.Log.With().Str("func", "attachPermissionScopesAndPolicies").Logger()

	apiScopes := make([]*model.AuthzScope, 0, len(params.ScopeIds))
	for _, scopeID := range params.ScopeIds {
		_, err := g.StorageProvider.AddPermissionScope(ctx, &schemas.PermissionScope{
			PermissionID: permissionID,
			ScopeID:      scopeID,
		})
		if err != nil {
			log.Debug().Err(err).Str("scope_id", scopeID).Msg("Failed to add permission scope")
			return nil, nil, err
		}
		scope, err := g.StorageProvider.GetScopeByID(ctx, scopeID)
		if err != nil {
			log.Debug().Err(err).Str("scope_id", scopeID).Msg("Failed to get scope by ID")
			return nil, nil, err
		}
		apiScopes = append(apiScopes, scope.AsAPIScope())
	}

	apiPolicies := make([]*model.AuthzPolicy, 0, len(params.PolicyIds))
	for _, policyID := range params.PolicyIds {
		_, err := g.StorageProvider.AddPermissionPolicy(ctx, &schemas.PermissionPolicy{
			PermissionID: permissionID,
			PolicyID:     policyID,
		})
		if err != nil {
			log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to add permission policy")
			return nil, nil, err
		}
		policy, err := g.StorageProvider.GetPolicyByID(ctx, policyID)
		if err != nil {
			log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to get policy by ID")
			return nil, nil, err
		}
		targets, err := g.StorageProvider.GetPolicyTargets(ctx, policyID)
		if err != nil {
			log.Debug().Err(err).Str("policy_id", policyID).Msg("Failed to get policy targets")
			return nil, nil, err
		}
		apiPolicies = append(apiPolicies, policy.AsAPIPolicy(targets))
	}

	return apiScopes, apiPolicies, nil
}
