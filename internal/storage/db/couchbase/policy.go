package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Policy).Insert(policy.ID, policy, &insertOpt)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// UpdatePolicy updates an existing authorization policy.
func (p *provider) UpdatePolicy(ctx context.Context, policy *schemas.Policy) (*schemas.Policy, error) {
	policy.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	policyMap := map[string]interface{}{}
	err = decoder.Decode(&policyMap)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(policyMap)
	params["_id"] = policy.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Policy, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// DeletePolicy deletes an authorization policy by ID.
// Returns an error if any permission_policy references this policy.
// Cascade-deletes associated policy targets.
func (p *provider) DeletePolicy(ctx context.Context, id string) error {
	// Check for permission_policy references
	params := make(map[string]interface{}, 1)
	params["policy_id"] = id
	query := fmt.Sprintf(`SELECT COUNT(*) as Total FROM %s.%s WHERE policy_id=$policy_id`, p.scopeName, schemas.Collections.PermissionPolicy)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	var totalDocs TotalDocs
	err = queryResult.One(&totalDocs)
	if err != nil {
		return err
	}
	if totalDocs.Total > 0 {
		return fmt.Errorf("cannot delete policy: %d permission_policy(s) reference it", totalDocs.Total)
	}
	// Cascade-delete policy targets
	deleteQuery := fmt.Sprintf(`DELETE FROM %s.%s WHERE policy_id=$policy_id`, p.scopeName, schemas.Collections.PolicyTarget)
	_, err = p.db.Query(deleteQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err = p.db.Collection(schemas.Collections.Policy).Remove(id, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyByID returns an authorization policy by its ID.
func (p *provider) GetPolicyByID(ctx context.Context, id string) (*schemas.Policy, error) {
	var policy *schemas.Policy
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT _id, name, description, type, logic, decision_strategy, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, schemas.Collections.Policy)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&policy)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// ListPolicies returns a paginated list of authorization policies.
func (p *provider) ListPolicies(ctx context.Context, pagination *model.Pagination) ([]*schemas.Policy, *model.Pagination, error) {
	policies := []*schemas.Policy{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Policy)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, name, description, type, logic, decision_strategy, created_at, updated_at FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.Policy)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var policy schemas.Policy
		err := queryResult.Row(&policy)
		if err != nil {
			log.Fatal(err)
		}
		policies = append(policies, &policy)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.PolicyTarget).Insert(target.ID, target, &insertOpt)
	if err != nil {
		return nil, err
	}
	return target, nil
}

// DeletePolicyTargetsByPolicyID removes all targets for a policy.
func (p *provider) DeletePolicyTargetsByPolicyID(ctx context.Context, policyID string) error {
	params := make(map[string]interface{}, 1)
	params["policy_id"] = policyID
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE policy_id=$policy_id`, p.scopeName, schemas.Collections.PolicyTarget)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyTargets returns all targets for a policy.
func (p *provider) GetPolicyTargets(ctx context.Context, policyID string) ([]*schemas.PolicyTarget, error) {
	targets := []*schemas.PolicyTarget{}
	params := make(map[string]interface{}, 1)
	params["policy_id"] = policyID
	query := fmt.Sprintf(`SELECT _id, policy_id, target_type, target_value, created_at FROM %s.%s WHERE policy_id=$policy_id`, p.scopeName, schemas.Collections.PolicyTarget)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var target schemas.PolicyTarget
		err := queryResult.Row(&target)
		if err != nil {
			log.Fatal(err)
		}
		targets = append(targets, &target)
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}
