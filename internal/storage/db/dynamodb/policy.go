package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddPolicy creates a new authorization policy.
func (p *provider) AddPolicy(ctx context.Context, policy *schemas.Policy) (*schemas.Policy, error) {
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}
	policy.Key = policy.ID
	policy.CreatedAt = time.Now().Unix()
	policy.UpdatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.Policy, policy); err != nil {
		return nil, err
	}
	return policy, nil
}

// UpdatePolicy updates an existing authorization policy.
func (p *provider) UpdatePolicy(ctx context.Context, policy *schemas.Policy) (*schemas.Policy, error) {
	policy.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Policy, "id", policy.ID, policy); err != nil {
		return nil, err
	}
	return policy, nil
}

// DeletePolicy deletes an authorization policy by ID.
// Returns an error if any permission_policy references this policy.
// Also cascade-deletes all policy_targets for this policy.
func (p *provider) DeletePolicy(ctx context.Context, id string) error {
	// Check for referencing permission_policies
	f := expression.Name("policy_id").Equal(expression.Value(id))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionPolicy, nil, &f)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return fmt.Errorf("cannot delete policy: %d permission_policy(ies) reference it", len(items))
	}
	// Cascade-delete policy targets
	if err := p.DeletePolicyTargetsByPolicyID(ctx, id); err != nil {
		return err
	}
	return p.deleteItemByHash(ctx, schemas.Collections.Policy, "id", id)
}

// GetPolicyByID returns an authorization policy by its ID.
func (p *provider) GetPolicyByID(ctx context.Context, id string) (*schemas.Policy, error) {
	var policy schemas.Policy
	if err := p.getItemByHash(ctx, schemas.Collections.Policy, "id", id, &policy); err != nil {
		return nil, err
	}
	if policy.ID == "" {
		return nil, errors.New("no document found")
	}
	return &policy, nil
}

// ListPolicies returns a paginated list of authorization policies.
func (p *provider) ListPolicies(ctx context.Context, pagination *model.Pagination) ([]*schemas.Policy, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var policies []*schemas.Policy

	count, err := p.scanCount(ctx, schemas.Collections.Policy, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.Policy, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var pol schemas.Policy
			if err := unmarshalItem(it, &pol); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				policies = append(policies, &pol)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return policies, paginationClone, nil
}

// AddPolicyTarget adds a target (role name or user ID) to a policy.
func (p *provider) AddPolicyTarget(ctx context.Context, target *schemas.PolicyTarget) (*schemas.PolicyTarget, error) {
	if target.ID == "" {
		target.ID = uuid.New().String()
	}
	target.Key = target.ID
	target.CreatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.PolicyTarget, target); err != nil {
		return nil, err
	}
	return target, nil
}

// DeletePolicyTargetsByPolicyID removes all targets for a policy.
func (p *provider) DeletePolicyTargetsByPolicyID(ctx context.Context, policyID string) error {
	f := expression.Name("policy_id").Equal(expression.Value(policyID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PolicyTarget, nil, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var target schemas.PolicyTarget
		if err := unmarshalItem(it, &target); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.PolicyTarget, "id", target.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetPolicyTargets returns all targets for a policy.
func (p *provider) GetPolicyTargets(ctx context.Context, policyID string) ([]*schemas.PolicyTarget, error) {
	f := expression.Name("policy_id").Equal(expression.Value(policyID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PolicyTarget, nil, &f)
	if err != nil {
		return nil, err
	}
	var targets []*schemas.PolicyTarget
	for _, it := range items {
		var target schemas.PolicyTarget
		if err := unmarshalItem(it, &target); err != nil {
			return nil, err
		}
		targets = append(targets, &target)
	}
	return targets, nil
}
