package schemas

import "github.com/authorizerdev/authorizer/internal/graph/model"

// Policy defines conditions for granting or denying access.
// Policies are the brain of the authorization model -- they determine WHO gets access.
// A policy has a Type (role-based, user-based, etc.) and Logic (positive=grant, negative=deny).
type Policy struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// Name is a unique human-readable identifier (e.g., "editors-policy").
	Name string `json:"name" gorm:"type:varchar(100);uniqueIndex" bson:"name" cql:"name" dynamo:"name"`
	// Description provides optional context about this policy.
	Description string `json:"description" gorm:"type:text" bson:"description" cql:"description" dynamo:"description"`
	// Type is the policy type discriminator: "role" or "user" (extensible to "client", "agent").
	// See constants.PolicyTypeRole, constants.PolicyTypeUser.
	Type string `json:"type" gorm:"type:varchar(50);index" bson:"type" cql:"type" dynamo:"type"`
	// Logic determines whether matching GRANTS or DENIES access.
	// "positive" = grant when matched, "negative" = deny when matched.
	// See constants.PolicyLogicPositive, constants.PolicyLogicNegative.
	Logic string `json:"logic" gorm:"type:varchar(10);default:positive" bson:"logic" cql:"logic" dynamo:"logic"`
	// DecisionStrategy controls how multiple targets within this policy are evaluated.
	// "affirmative" = any target match grants, "unanimous" = all targets must match.
	// See constants.DecisionStrategyAffirmative, constants.DecisionStrategyUnanimous.
	DecisionStrategy string `json:"decision_strategy" gorm:"type:varchar(20);default:affirmative" bson:"decision_strategy" cql:"decision_strategy" dynamo:"decision_strategy"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	// UpdatedAt is the unix timestamp of last update.
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// PolicyTarget specifies who/what a policy applies to.
// For a role-based policy, targets are role names. For a user-based policy, targets are user IDs.
type PolicyTarget struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// PolicyID is the foreign key to the parent Policy.
	PolicyID string `json:"policy_id" gorm:"type:char(36);index;uniqueIndex:idx_pt_unique" bson:"policy_id" cql:"policy_id" dynamo:"policy_id"`
	// TargetType describes what kind of target this is: "role" or "user"
	// (extensible to "client", "agent").
	TargetType string `json:"target_type" gorm:"type:varchar(50);uniqueIndex:idx_pt_unique" bson:"target_type" cql:"target_type" dynamo:"target_type"`
	// TargetValue is the role name or user/client/agent ID this target matches.
	TargetValue string `json:"target_value" gorm:"type:varchar(256);uniqueIndex:idx_pt_unique" bson:"target_value" cql:"target_value" dynamo:"target_value"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
}

// AsAPIPolicy converts a storage Policy and its targets to the GraphQL API model.
func (p *Policy) AsAPIPolicy(targets []*PolicyTarget) *model.AuthzPolicy {
	apiTargets := make([]*model.AuthzPolicyTarget, len(targets))
	for i, t := range targets {
		apiTargets[i] = &model.AuthzPolicyTarget{
			ID:          t.ID,
			TargetType:  t.TargetType,
			TargetValue: t.TargetValue,
		}
	}
	return &model.AuthzPolicy{
		ID:               p.ID,
		Name:             p.Name,
		Description:      &p.Description,
		Type:             p.Type,
		Logic:            p.Logic,
		DecisionStrategy: p.DecisionStrategy,
		Targets:          apiTargets,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}
