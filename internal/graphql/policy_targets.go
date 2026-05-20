package graphql

import (
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// validatePolicyTargets enforces that every target on a policy is consistent
// with the policy's type and references a real value:
//
//   - target_type must equal policyType (role or user) so that the storage row
//     and the evaluator agree on how to match it.
//   - target_value must be non-empty after trimming.
//   - For role targets, target_value must be one of the configured ROLES so
//     that policies cannot be silently dead — a typo'd role would evaluate to
//     "no match" forever.
//
// User targets are not checked against the users table here: that lookup is
// per-target and would race with deletes, so we only enforce non-emptiness
// and let the evaluator no-op on missing IDs.
func validatePolicyTargets(policyType string, targets []*model.PolicyTargetInput, configRoles []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("at least one policy target is required")
	}

	allowedRoles := make(map[string]bool, len(configRoles))
	for _, r := range configRoles {
		allowedRoles[r] = true
	}

	for i, t := range targets {
		if t.TargetType != policyType {
			return fmt.Errorf("target %d: target_type %q does not match policy type %q", i, t.TargetType, policyType)
		}
		value := strings.TrimSpace(t.TargetValue)
		if value == "" {
			return fmt.Errorf("target %d: target_value is required", i)
		}
		if policyType == constants.PolicyTypeRole && !allowedRoles[value] {
			return fmt.Errorf("target %d: role %q is not in configured ROLES", i, value)
		}
	}
	return nil
}
