package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

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
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&policy)
	if res.Error != nil {
		return nil, res.Error
	}
	return policy, nil
}

// UpdatePolicy updates an existing authorization policy.
func (p *provider) UpdatePolicy(ctx context.Context, policy *schemas.Policy) (*schemas.Policy, error) {
	policy.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&policy)
	if result.Error != nil {
		return nil, result.Error
	}
	return policy, nil
}

// DeletePolicy deletes an authorization policy by ID.
// Returns an error if any permission_policy references this policy.
// Cascade-deletes associated policy targets.
func (p *provider) DeletePolicy(ctx context.Context, id string) error {
	var count int64
	p.db.Model(&schemas.PermissionPolicy{}).Where("policy_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("cannot delete policy: %d permission_policy(s) reference it", count)
	}
	// Cascade-delete policy targets
	result := p.db.Where("policy_id = ?", id).Delete(&schemas.PolicyTarget{})
	if result.Error != nil {
		return result.Error
	}
	result = p.db.Where("id = ?", id).Delete(&schemas.Policy{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetPolicyByID returns an authorization policy by its ID.
func (p *provider) GetPolicyByID(ctx context.Context, id string) (*schemas.Policy, error) {
	var policy schemas.Policy
	result := p.db.Where("id = ?", id).First(&policy)
	if result.Error != nil {
		return nil, result.Error
	}
	return &policy, nil
}

// ListPolicies returns a paginated list of authorization policies.
func (p *provider) ListPolicies(ctx context.Context, pagination *model.Pagination) ([]*schemas.Policy, *model.Pagination, error) {
	var policies []*schemas.Policy
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&policies)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.Model(&schemas.Policy{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total
	return policies, paginationClone, nil
}

// AddPolicyTarget adds a target (role name or user ID) to a policy.
func (p *provider) AddPolicyTarget(ctx context.Context, target *schemas.PolicyTarget) (*schemas.PolicyTarget, error) {
	if target.ID == "" {
		target.ID = uuid.New().String()
	}
	target.Key = target.ID
	target.CreatedAt = time.Now().Unix()
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&target)
	if res.Error != nil {
		return nil, res.Error
	}
	return target, nil
}

// DeletePolicyTargetsByPolicyID removes all targets for a policy.
func (p *provider) DeletePolicyTargetsByPolicyID(ctx context.Context, policyID string) error {
	result := p.db.Where("policy_id = ?", policyID).Delete(&schemas.PolicyTarget{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetPolicyTargets returns all targets for a policy.
func (p *provider) GetPolicyTargets(ctx context.Context, policyID string) ([]*schemas.PolicyTarget, error) {
	var targets []*schemas.PolicyTarget
	result := p.db.Where("policy_id = ?", policyID).Find(&targets)
	if result.Error != nil {
		return nil, result.Error
	}
	return targets, nil
}
