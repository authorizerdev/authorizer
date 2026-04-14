package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
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
	collection, _ := p.db.Collection(ctx, schemas.Collections.Policy)
	meta, err := collection.CreateDocument(ctx, policy)
	if err != nil {
		return nil, err
	}
	policy.Key = meta.Key
	policy.ID = meta.ID.String()
	return policy, nil
}

// UpdatePolicy updates an existing authorization policy.
func (p *provider) UpdatePolicy(ctx context.Context, policy *schemas.Policy) (*schemas.Policy, error) {
	policy.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.Policy)
	meta, err := collection.UpdateDocument(ctx, policy.Key, policy)
	if err != nil {
		return nil, err
	}
	policy.Key = meta.Key
	policy.ID = meta.ID.String()
	return policy, nil
}

// DeletePolicy deletes an authorization policy by ID.
// Returns an error if any permission_policy references this policy.
// Cascade-deletes associated policy targets.
func (p *provider) DeletePolicy(ctx context.Context, id string) error {
	// Check for referencing permission_policies
	countQuery := fmt.Sprintf("FOR d IN %s FILTER d.policy_id == @policy_id COLLECT WITH COUNT INTO length RETURN length", schemas.Collections.PermissionPolicy)
	cursor, err := p.db.Query(ctx, countQuery, map[string]interface{}{
		"policy_id": id,
	})
	if err != nil {
		return err
	}
	defer cursor.Close()
	var count int64
	if cursor.HasMore() {
		if _, err := cursor.ReadDocument(ctx, &count); err != nil {
			return err
		}
	}
	if count > 0 {
		return fmt.Errorf("cannot delete policy: %d permission_policy(s) reference it", count)
	}

	// Cascade-delete policy targets
	deleteTargetsQuery := fmt.Sprintf("FOR d IN %s FILTER d.policy_id == @policy_id REMOVE d IN %s", schemas.Collections.PolicyTarget, schemas.Collections.PolicyTarget)
	targetCursor, err := p.db.Query(ctx, deleteTargetsQuery, map[string]interface{}{
		"policy_id": id,
	})
	if err != nil {
		return err
	}
	defer targetCursor.Close()

	// Find the document key for this policy
	policy, err := p.GetPolicyByID(ctx, id)
	if err != nil {
		return err
	}
	collection, _ := p.db.Collection(ctx, schemas.Collections.Policy)
	_, err = collection.RemoveDocument(ctx, policy.Key)
	return err
}

// GetPolicyByID returns an authorization policy by its ID.
func (p *provider) GetPolicyByID(ctx context.Context, id string) (*schemas.Policy, error) {
	var policy *schemas.Policy
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id RETURN d", schemas.Collections.Policy)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if policy == nil {
				return nil, fmt.Errorf("policy not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &policy)
		if err != nil {
			return nil, err
		}
	}
	return policy, nil
}

// ListPolicies returns a paginated list of authorization policies.
func (p *provider) ListPolicies(ctx context.Context, pagination *model.Pagination) ([]*schemas.Policy, *model.Pagination, error) {
	policies := []*schemas.Policy{}
	query := fmt.Sprintf("FOR d IN %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Policy, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var policy *schemas.Policy
		meta, err := cursor.ReadDocument(ctx, &policy)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			policies = append(policies, policy)
		}
	}
	return policies, paginationClone, nil
}

// AddPolicyTarget adds a target (role name or user ID) to a policy.
func (p *provider) AddPolicyTarget(ctx context.Context, target *schemas.PolicyTarget) (*schemas.PolicyTarget, error) {
	if target.ID == "" {
		target.ID = uuid.New().String()
	}
	target.Key = target.ID
	target.CreatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.PolicyTarget)
	meta, err := collection.CreateDocument(ctx, target)
	if err != nil {
		return nil, err
	}
	target.Key = meta.Key
	target.ID = meta.ID.String()
	return target, nil
}

// DeletePolicyTargetsByPolicyID removes all targets for a policy.
func (p *provider) DeletePolicyTargetsByPolicyID(ctx context.Context, policyID string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.policy_id == @policy_id REMOVE d IN %s", schemas.Collections.PolicyTarget, schemas.Collections.PolicyTarget)
	cursor, err := p.db.Query(ctx, query, map[string]interface{}{
		"policy_id": policyID,
	})
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetPolicyTargets returns all targets for a policy.
func (p *provider) GetPolicyTargets(ctx context.Context, policyID string) ([]*schemas.PolicyTarget, error) {
	targets := []*schemas.PolicyTarget{}
	query := fmt.Sprintf("FOR d IN %s FILTER d.policy_id == @policy_id RETURN d", schemas.Collections.PolicyTarget)
	cursor, err := p.db.Query(ctx, query, map[string]interface{}{
		"policy_id": policyID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		var target *schemas.PolicyTarget
		if _, err := cursor.ReadDocument(ctx, &target); arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}
