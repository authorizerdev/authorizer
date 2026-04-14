package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UpdatePolicy is the method to update an existing authorization policy.
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdatePolicy(ctx context.Context, params *model.UpdatePolicyInput) (*model.AuthzPolicy, error) {
	log := g.Log.With().Str("func", "UpdatePolicy").Logger()
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
		return nil, fmt.Errorf("policy ID is required")
	}

	policy, err := g.StorageProvider.GetPolicyByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get policy by ID")
		return nil, err
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			return nil, fmt.Errorf("policy name cannot be empty")
		}
		policy.Name = name
	}
	if params.Description != nil {
		policy.Description = *params.Description
	}
	if params.Logic != nil {
		policy.Logic = *params.Logic
	}
	if params.DecisionStrategy != nil {
		policy.DecisionStrategy = *params.DecisionStrategy
	}

	policy, err = g.StorageProvider.UpdatePolicy(ctx, policy)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update policy")
		return nil, err
	}

	// Replace targets if provided
	var targets []*schemas.PolicyTarget
	if params.Targets != nil {
		err = g.StorageProvider.DeletePolicyTargetsByPolicyID(ctx, policy.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete existing policy targets")
			return nil, err
		}
		for _, t := range params.Targets {
			target, err := g.StorageProvider.AddPolicyTarget(ctx, &schemas.PolicyTarget{
				PolicyID:    policy.ID,
				TargetType:  t.TargetType,
				TargetValue: t.TargetValue,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Failed to add policy target")
				return nil, err
			}
			targets = append(targets, target)
		}
	} else {
		targets, err = g.StorageProvider.GetPolicyTargets(ctx, policy.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get policy targets")
			return nil, err
		}
	}

	go g.AuthorizationProvider.InvalidateCache(ctx, "authz:")

	return policy.AsAPIPolicy(targets), nil
}
