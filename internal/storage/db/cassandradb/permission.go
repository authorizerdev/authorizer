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

// AddPermission creates a new authorization permission.
func (p *provider) AddPermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}
	permission.Key = permission.ID
	permission.CreatedAt = time.Now().Unix()
	permission.UpdatedAt = time.Now().Unix()
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, name, description, resource_id, decision_strategy, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.Permission)
	err := p.db.Query(insertQuery, permission.ID, permission.Name, permission.Description, permission.ResourceID, permission.DecisionStrategy, permission.CreatedAt, permission.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// UpdatePermission updates an existing authorization permission.
func (p *provider) UpdatePermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	permission.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(permission)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	permissionMap := map[string]interface{}{}
	err = decoder.Decode(&permissionMap)
	if err != nil {
		return nil, err
	}
	convertMapValues(permissionMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range permissionMap {
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
	updateValues = append(updateValues, permission.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Permission, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// DeletePermission deletes an authorization permission by ID.
// Cascade-deletes associated permission_scopes and permission_policies.
func (p *provider) DeletePermission(ctx context.Context, id string) error {
	// Cascade-delete permission_scopes
	getScopesQuery := fmt.Sprintf("SELECT id FROM %s WHERE permission_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PermissionScope)
	scanner := p.db.Query(getScopesQuery, id).Iter().Scanner()
	var scopeIDs []string
	for scanner.Next() {
		var scopeID string
		err := scanner.Scan(&scopeID)
		if err != nil {
			return err
		}
		scopeIDs = append(scopeIDs, scopeID)
	}
	if len(scopeIDs) > 0 {
		placeholders := strings.Repeat("?,", len(scopeIDs))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(scopeIDs))
		for i, sid := range scopeIDs {
			deleteValues[i] = sid
		}
		deleteScopesQuery := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.PermissionScope, placeholders)
		err := p.db.Query(deleteScopesQuery, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}
	// Cascade-delete permission_policies
	getPoliciesQuery := fmt.Sprintf("SELECT id FROM %s WHERE permission_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PermissionPolicy)
	scanner = p.db.Query(getPoliciesQuery, id).Iter().Scanner()
	var policyIDs []string
	for scanner.Next() {
		var policyID string
		err := scanner.Scan(&policyID)
		if err != nil {
			return err
		}
		policyIDs = append(policyIDs, policyID)
	}
	if len(policyIDs) > 0 {
		placeholders := strings.Repeat("?,", len(policyIDs))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(policyIDs))
		for i, pid := range policyIDs {
			deleteValues[i] = pid
		}
		deletePoliciesQuery := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.PermissionPolicy, placeholders)
		err := p.db.Query(deletePoliciesQuery, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}
	// Delete the permission itself
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Permission)
	err := p.db.Query(query, id).Exec()
	if err != nil {
		return err
	}
	return nil
}

// GetPermissionByID returns an authorization permission by its ID.
func (p *provider) GetPermissionByID(ctx context.Context, id string) (*schemas.Permission, error) {
	var permission schemas.Permission
	query := fmt.Sprintf("SELECT id, name, description, resource_id, decision_strategy, created_at, updated_at FROM %s WHERE id = ? LIMIT 1",
		KeySpace+"."+schemas.Collections.Permission)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(
		&permission.ID, &permission.Name, &permission.Description, &permission.ResourceID, &permission.DecisionStrategy, &permission.CreatedAt, &permission.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// ListPermissions returns a paginated list of authorization permissions.
func (p *provider) ListPermissions(ctx context.Context, pagination *model.Pagination) ([]*schemas.Permission, *model.Pagination, error) {
	permissions := []*schemas.Permission{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", KeySpace+"."+schemas.Collections.Permission)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	query := fmt.Sprintf("SELECT id, name, description, resource_id, decision_strategy, created_at, updated_at FROM %s LIMIT %d",
		KeySpace+"."+schemas.Collections.Permission, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var permission schemas.Permission
			err := scanner.Scan(&permission.ID, &permission.Name, &permission.Description, &permission.ResourceID, &permission.DecisionStrategy, &permission.CreatedAt, &permission.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			permissions = append(permissions, &permission)
		}
		counter++
	}
	return permissions, paginationClone, nil
}

// AddPermissionScope links a scope to a permission.
func (p *provider) AddPermissionScope(ctx context.Context, ps *schemas.PermissionScope) (*schemas.PermissionScope, error) {
	if ps.ID == "" {
		ps.ID = uuid.New().String()
	}
	ps.Key = ps.ID
	ps.CreatedAt = time.Now().Unix()
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, permission_id, scope_id, created_at) VALUES (?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.PermissionScope)
	err := p.db.Query(insertQuery, ps.ID, ps.PermissionID, ps.ScopeID, ps.CreatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return ps, nil
}

// DeletePermissionScopesByPermissionID removes all scope links for a permission.
func (p *provider) DeletePermissionScopesByPermissionID(ctx context.Context, permissionID string) error {
	getQuery := fmt.Sprintf("SELECT id FROM %s WHERE permission_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PermissionScope)
	scanner := p.db.Query(getQuery, permissionID).Iter().Scanner()
	var ids []string
	for scanner.Next() {
		var id string
		err := scanner.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		placeholders := strings.Repeat("?,", len(ids))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(ids))
		for i, id := range ids {
			deleteValues[i] = id
		}
		query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.PermissionScope, placeholders)
		err := p.db.Query(query, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPermissionScopes returns all scope links for a permission.
func (p *provider) GetPermissionScopes(ctx context.Context, permissionID string) ([]*schemas.PermissionScope, error) {
	scopes := []*schemas.PermissionScope{}
	query := fmt.Sprintf("SELECT id, permission_id, scope_id, created_at FROM %s WHERE permission_id = ? ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.PermissionScope)
	scanner := p.db.Query(query, permissionID).Iter().Scanner()
	for scanner.Next() {
		var ps schemas.PermissionScope
		err := scanner.Scan(&ps.ID, &ps.PermissionID, &ps.ScopeID, &ps.CreatedAt)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, &ps)
	}
	return scopes, nil
}

// AddPermissionPolicy links a policy to a permission.
func (p *provider) AddPermissionPolicy(ctx context.Context, pp *schemas.PermissionPolicy) (*schemas.PermissionPolicy, error) {
	if pp.ID == "" {
		pp.ID = uuid.New().String()
	}
	pp.Key = pp.ID
	pp.CreatedAt = time.Now().Unix()
	insertQuery := fmt.Sprintf("INSERT INTO %s (id, permission_id, policy_id, created_at) VALUES (?, ?, ?, ?)",
		KeySpace+"."+schemas.Collections.PermissionPolicy)
	err := p.db.Query(insertQuery, pp.ID, pp.PermissionID, pp.PolicyID, pp.CreatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return pp, nil
}

// DeletePermissionPoliciesByPermissionID removes all policy links for a permission.
func (p *provider) DeletePermissionPoliciesByPermissionID(ctx context.Context, permissionID string) error {
	getQuery := fmt.Sprintf("SELECT id FROM %s WHERE permission_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.PermissionPolicy)
	scanner := p.db.Query(getQuery, permissionID).Iter().Scanner()
	var ids []string
	for scanner.Next() {
		var id string
		err := scanner.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		placeholders := strings.Repeat("?,", len(ids))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(ids))
		for i, id := range ids {
			deleteValues[i] = id
		}
		query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.PermissionPolicy, placeholders)
		err := p.db.Query(query, deleteValues...).Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPermissionPolicies returns all policy links for a permission.
func (p *provider) GetPermissionPolicies(ctx context.Context, permissionID string) ([]*schemas.PermissionPolicy, error) {
	policies := []*schemas.PermissionPolicy{}
	query := fmt.Sprintf("SELECT id, permission_id, policy_id, created_at FROM %s WHERE permission_id = ? ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.PermissionPolicy)
	scanner := p.db.Query(query, permissionID).Iter().Scanner()
	for scanner.Next() {
		var pp schemas.PermissionPolicy
		err := scanner.Scan(&pp.ID, &pp.PermissionID, &pp.PolicyID, &pp.CreatedAt)
		if err != nil {
			return nil, err
		}
		policies = append(policies, &pp)
	}
	return policies, nil
}

// GetPermissionsForResourceScope returns all permissions (with their policies and targets)
// that match a given resource name and scope name. This is the hot-path query used by
// the evaluation engine. Uses sequential queries since Cassandra does not support JOINs.
func (p *provider) GetPermissionsForResourceScope(ctx context.Context, resourceName string, scopeName string) ([]*schemas.PermissionWithPolicies, error) {
	// 1. Find resource by name
	var resourceID string
	resourceQuery := fmt.Sprintf("SELECT id FROM %s WHERE name = ? LIMIT 1 ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.Resource)
	err := p.db.Query(resourceQuery, resourceName).Consistency(gocql.One).Scan(&resourceID)
	if err != nil {
		return nil, err
	}

	// 2. Find scope by name
	var scopeID string
	scopeQuery := fmt.Sprintf("SELECT id FROM %s WHERE name = ? LIMIT 1 ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.Scope)
	err = p.db.Query(scopeQuery, scopeName).Consistency(gocql.One).Scan(&scopeID)
	if err != nil {
		return nil, err
	}

	// 3. Find permissions for this resource
	permQuery := fmt.Sprintf("SELECT id, name, decision_strategy FROM %s WHERE resource_id = ? ALLOW FILTERING",
		KeySpace+"."+schemas.Collections.Permission)
	permScanner := p.db.Query(permQuery, resourceID).Iter().Scanner()

	type permInfo struct {
		ID               string
		Name             string
		DecisionStrategy string
	}
	var permissions []permInfo
	for permScanner.Next() {
		var pi permInfo
		err := permScanner.Scan(&pi.ID, &pi.Name, &pi.DecisionStrategy)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, pi)
	}

	if len(permissions) == 0 {
		return nil, nil
	}

	// 4. For each permission, check if it has the requested scope
	var result []*schemas.PermissionWithPolicies

	for _, perm := range permissions {
		var scopeCount int64
		scopeCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE permission_id = ? AND scope_id = ? ALLOW FILTERING",
			KeySpace+"."+schemas.Collections.PermissionScope)
		err := p.db.Query(scopeCountQuery, perm.ID, scopeID).Consistency(gocql.One).Scan(&scopeCount)
		if err != nil {
			return nil, err
		}
		if scopeCount == 0 {
			continue
		}

		// 5. Find permission_policies for this permission
		ppQuery := fmt.Sprintf("SELECT policy_id FROM %s WHERE permission_id = ? ALLOW FILTERING",
			KeySpace+"."+schemas.Collections.PermissionPolicy)
		ppScanner := p.db.Query(ppQuery, perm.ID).Iter().Scanner()
		var policyIDs []string
		for ppScanner.Next() {
			var policyID string
			err := ppScanner.Scan(&policyID)
			if err != nil {
				return nil, err
			}
			policyIDs = append(policyIDs, policyID)
		}

		if len(policyIDs) == 0 {
			continue
		}

		// 6. For each policy, resolve the policy and its targets
		var policiesWithTargets []schemas.PolicyWithTargets
		for _, policyID := range policyIDs {
			var policy schemas.Policy
			policyQuery := fmt.Sprintf("SELECT id, name, type, logic, decision_strategy FROM %s WHERE id = ? LIMIT 1",
				KeySpace+"."+schemas.Collections.Policy)
			err := p.db.Query(policyQuery, policyID).Consistency(gocql.One).Scan(
				&policy.ID, &policy.Name, &policy.Type, &policy.Logic, &policy.DecisionStrategy)
			if err != nil {
				return nil, err
			}

			// Get targets for this policy
			targetQuery := fmt.Sprintf("SELECT target_type, target_value FROM %s WHERE policy_id = ? ALLOW FILTERING",
				KeySpace+"."+schemas.Collections.PolicyTarget)
			targetScanner := p.db.Query(targetQuery, policyID).Iter().Scanner()
			var targets []schemas.PolicyTargetView
			for targetScanner.Next() {
				var tv schemas.PolicyTargetView
				err := targetScanner.Scan(&tv.TargetType, &tv.TargetValue)
				if err != nil {
					return nil, err
				}
				targets = append(targets, tv)
			}

			policiesWithTargets = append(policiesWithTargets, schemas.PolicyWithTargets{
				PolicyID:         policy.ID,
				PolicyName:       policy.Name,
				Type:             policy.Type,
				Logic:            policy.Logic,
				DecisionStrategy: policy.DecisionStrategy,
				Targets:          targets,
			})
		}

		result = append(result, &schemas.PermissionWithPolicies{
			PermissionID:     perm.ID,
			PermissionName:   perm.Name,
			DecisionStrategy: perm.DecisionStrategy,
			Policies:         policiesWithTargets,
		})
	}

	return result, nil
}
