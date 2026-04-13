package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
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
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, name, description, type, logic, decision_strategy, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.Policy)
	err := p.db.Query(insertQuery, policy.ID, policy.Name, policy.Description, policy.Type, policy.Logic, policy.DecisionStrategy, policy.CreatedAt, policy.UpdatedAt).Exec()
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
	convertMapValues(policyMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range policyMap {
		if key == "_id" || key == "_key" || key == "id" || key == "key" {
			continue
		}
		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}
		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	updateValues = append(updateValues, policy.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Policy, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// DeletePolicy deletes an authorization policy by ID.
// Returns an error if any permission_policy references this policy.
// Cascade-deletes associated policy targets.
func (p *provider) DeletePolicy(ctx context.Context, id string) error {
	// Check for referencing permission_policies
	var count int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE policy_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PermissionPolicy)
	err := p.db.Query(countQuery, id).Consistency(gocql.One).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete policy: %d permission_policy(s) reference it", count)
	}
	// Cascade-delete policy targets
	getTargetsQuery := fmt.Sprintf("SELECT id FROM %s WHERE policy_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PolicyTarget)
	scanner := p.db.Query(getTargetsQuery, id).Iter().Scanner()
	var targetIDs []string
	for scanner.Next() {
		var targetID string
		err = scanner.Scan(&targetID)
		if err != nil {
			return err
		}
		targetIDs = append(targetIDs, targetID)
	}
	if len(targetIDs) > 0 {
		placeholders := strings.Repeat("?,", len(targetIDs))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(targetIDs))
		for i, tid := range targetIDs {
			deleteValues[i] = tid
		}
		deleteTargetsQuery := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.PolicyTarget, placeholders)
		err = p.db.Query(deleteTargetsQuery, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}
	// Delete the policy itself
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Policy)
	err = p.db.Query(query, id).Exec()
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyByID returns an authorization policy by its ID.
func (p *provider) GetPolicyByID(ctx context.Context, id string) (*schemas.Policy, error) {
	var policy schemas.Policy
	query := fmt.Sprintf("SELECT id, name, description, type, logic, decision_strategy, created_at, updated_at FROM %s WHERE id = ? LIMIT 1",
		KeySpace+"."+schemas.Collections.Policy)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(
		&policy.ID, &policy.Name, &policy.Description, &policy.Type, &policy.Logic, &policy.DecisionStrategy, &policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// ListPolicies returns a paginated list of authorization policies.
func (p *provider) ListPolicies(ctx context.Context, pagination *model.Pagination) ([]*schemas.Policy, *model.Pagination, error) {
	policies := []*schemas.Policy{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.Policy)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	query := fmt.Sprintf("SELECT id, name, description, type, logic, decision_strategy, created_at, updated_at FROM %s LIMIT %d",
		KeySpace+"."+schemas.Collections.Policy, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var policy schemas.Policy
			err := scanner.Scan(&policy.ID, &policy.Name, &policy.Description, &policy.Type, &policy.Logic, &policy.DecisionStrategy, &policy.CreatedAt, &policy.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			policies = append(policies, &policy)
		}
		counter++
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
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, policy_id, target_type, target_value, created_at) VALUES (?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.PolicyTarget)
	err := p.db.Query(insertQuery, target.ID, target.PolicyID, target.TargetType, target.TargetValue, target.CreatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return target, nil
}

// DeletePolicyTargetsByPolicyID removes all targets for a policy.
func (p *provider) DeletePolicyTargetsByPolicyID(ctx context.Context, policyID string) error {
	getTargetsQuery := fmt.Sprintf("SELECT id FROM %s WHERE policy_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PolicyTarget)
	scanner := p.db.Query(getTargetsQuery, policyID).Iter().Scanner()
	var targetIDs []string
	for scanner.Next() {
		var targetID string
		err := scanner.Scan(&targetID)
		if err != nil {
			return err
		}
		targetIDs = append(targetIDs, targetID)
	}
	if len(targetIDs) > 0 {
		placeholders := strings.Repeat("?,", len(targetIDs))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(targetIDs))
		for i, tid := range targetIDs {
			deleteValues[i] = tid
		}
		query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.PolicyTarget, placeholders)
		err := p.db.Query(query, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPolicyTargets returns all targets for a policy.
func (p *provider) GetPolicyTargets(ctx context.Context, policyID string) ([]*schemas.PolicyTarget, error) {
	targets := []*schemas.PolicyTarget{}
	query := fmt.Sprintf("SELECT id, policy_id, target_type, target_value, created_at FROM %s WHERE policy_id = ? ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.PolicyTarget)
	scanner := p.db.Query(query, policyID).Iter().Scanner()
	for scanner.Next() {
		var target schemas.PolicyTarget
		err := scanner.Scan(&target.ID, &target.PolicyID, &target.TargetType, &target.TargetValue, &target.CreatedAt)
		if err != nil {
			return nil, err
		}
		targets = append(targets, &target)
	}
	return targets, nil
}
