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
	if len(name) > 100 {
		return nil, fmt.Errorf("invalid name: must be 100 characters or fewer")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return nil, fmt.Errorf("invalid name: must contain only letters, digits, hyphens, and underscores")
		}
	}

	policyType := strings.TrimSpace(params.Type)
	if policyType == "" {
		return nil, fmt.Errorf("policy type is required")
	}
	validPolicyTypes := map[string]bool{"role": true, "user": true}
	if !validPolicyTypes[policyType] {
		return nil, fmt.Errorf("invalid policy type: must be 'role' or 'user'")
	}

	description := ""
	if params.Description != nil {
		description = *params.Description
	}

	logic := "positive"
	if params.Logic != nil {
		logic = *params.Logic
	}
	if logic != "positive" && logic != "negative" {
		return nil, fmt.Errorf("invalid policy logic: must be 'positive' or 'negative'")
	}

	decisionStrategy := "affirmative"
	if params.DecisionStrategy != nil {
		decisionStrategy = *params.DecisionStrategy
	}
	if decisionStrategy != "affirmative" && decisionStrategy != "unanimous" {
		return nil, fmt.Errorf("invalid decision strategy: must be 'affirmative' or 'unanimous'")
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

	g.AuthorizationProvider.InvalidateCache(context.Background(), "authz:")

	return policy.AsAPIPolicy(targets), nil
}
