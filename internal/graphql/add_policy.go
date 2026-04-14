package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AddPolicy is the method to create a new authorization policy with targets.
// Permissions: authorizer:admin
func (g *graphqlProvider) AddPolicy(ctx context.Context, params *model.AddPolicyInput) (*model.AuthzPolicy, error) {
	log := g.Log.With().Str("func", "AddPolicy").Logger()
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
		return nil, fmt.Errorf("policy name is required")
	}

	policyType := strings.TrimSpace(params.Type)
	if policyType == "" {
		return nil, fmt.Errorf("policy type is required")
	}

	description := ""
	if params.Description != nil {
		description = *params.Description
	}

	logic := "positive"
	if params.Logic != nil {
		logic = *params.Logic
	}

	decisionStrategy := "affirmative"
	if params.DecisionStrategy != nil {
		decisionStrategy = *params.DecisionStrategy
	}

	policy, err := g.StorageProvider.AddPolicy(ctx, &schemas.Policy{
		Name:             name,
		Description:      description,
		Type:             policyType,
		Logic:            logic,
		DecisionStrategy: decisionStrategy,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add policy")
		return nil, err
	}

	// Create policy targets
	var targets []*schemas.PolicyTarget
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

	go g.AuthorizationProvider.InvalidateCache(ctx, "authz:")

	return policy.AsAPIPolicy(targets), nil
}
