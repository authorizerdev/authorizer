package graphql

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
)

func TestValidatePolicyTargets(t *testing.T) {
	roles := []string{"admin", "editor", "viewer"}

	t.Run("rejects empty targets", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeRole, nil, roles)
		assert.Error(t, err)
	})

	t.Run("rejects target_type mismatch", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeRole, []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeUser, TargetValue: "admin"},
		}, roles)
		assert.ErrorContains(t, err, "does not match policy type")
	})

	t.Run("rejects empty target_value", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeRole, []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeRole, TargetValue: "  "},
		}, roles)
		assert.ErrorContains(t, err, "target_value is required")
	})

	t.Run("rejects role not in configured ROLES", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeRole, []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeRole, TargetValue: "ghost"},
		}, roles)
		assert.ErrorContains(t, err, "not in configured ROLES")
	})

	t.Run("accepts role in configured ROLES", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeRole, []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeRole, TargetValue: "admin"},
			{TargetType: constants.TargetTypeRole, TargetValue: "editor"},
		}, roles)
		assert.NoError(t, err)
	})

	t.Run("user targets are not checked against ROLES", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeUser, []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeUser, TargetValue: "6f1a2b3c-4d5e-6f70-8a9b-0c1d2e3f4a5b"},
		}, roles)
		assert.NoError(t, err)
	})

	t.Run("user targets still require non-empty value", func(t *testing.T) {
		err := validatePolicyTargets(constants.PolicyTypeUser, []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeUser, TargetValue: ""},
		}, roles)
		assert.ErrorContains(t, err, "target_value is required")
	})
}
