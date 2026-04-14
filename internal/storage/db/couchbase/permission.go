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

// AddPermission creates a new authorization permission.
func (p *provider) AddPermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}
	permission.Key = permission.ID
	permission.CreatedAt = time.Now().Unix()
	permission.UpdatedAt = time.Now().Unix()
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Permission).Insert(permission.ID, permission, &insertOpt)
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
	updateFields, params := GetSetFields(permissionMap)
	params["_id"] = permission.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Permission, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// DeletePermission deletes an authorization permission by ID.
// Cascade-deletes associated permission_scopes and permission_policies.
func (p *provider) DeletePermission(ctx context.Context, id string) error {
	params := make(map[string]interface{}, 1)
	params["permission_id"] = id
	// Cascade-delete permission_scopes
	scopeQuery := fmt.Sprintf(`DELETE FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionScope)
	_, err := p.db.Query(scopeQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	// Cascade-delete permission_policies
	policyQuery := fmt.Sprintf(`DELETE FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionPolicy)
	_, err = p.db.Query(policyQuery, &gocb.QueryOptions{
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
	_, err = p.db.Collection(schemas.Collections.Permission).Remove(id, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetPermissionByID returns an authorization permission by its ID.
func (p *provider) GetPermissionByID(ctx context.Context, id string) (*schemas.Permission, error) {
	var permission *schemas.Permission
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT _id, name, description, resource_id, decision_strategy, created_at, updated_at FROM %s.%s WHERE _id=$_id LIMIT 1`, p.scopeName, schemas.Collections.Permission)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&permission)
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// ListPermissions returns a paginated list of authorization permissions.
func (p *provider) ListPermissions(ctx context.Context, pagination *model.Pagination) ([]*schemas.Permission, *model.Pagination, error) {
	permissions := []*schemas.Permission{}
	paginationClone := pagination
	params := make(map[string]interface{}, 1)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Permission)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT _id, name, description, resource_id, decision_strategy, created_at, updated_at FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", p.scopeName, schemas.Collections.Permission)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var permission schemas.Permission
		err := queryResult.Row(&permission)
		if err != nil {
			log.Fatal(err)
		}
		permissions = append(permissions, &permission)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.PermissionScope).Insert(ps.ID, ps, &insertOpt)
	if err != nil {
		return nil, err
	}
	return ps, nil
}

// DeletePermissionScopesByPermissionID removes all scope links for a permission.
func (p *provider) DeletePermissionScopesByPermissionID(ctx context.Context, permissionID string) error {
	params := make(map[string]interface{}, 1)
	params["permission_id"] = permissionID
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionScope)
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

// GetPermissionScopes returns all scope links for a permission.
func (p *provider) GetPermissionScopes(ctx context.Context, permissionID string) ([]*schemas.PermissionScope, error) {
	scopes := []*schemas.PermissionScope{}
	params := make(map[string]interface{}, 1)
	params["permission_id"] = permissionID
	query := fmt.Sprintf(`SELECT _id, permission_id, scope_id, created_at FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionScope)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var ps schemas.PermissionScope
		err := queryResult.Row(&ps)
		if err != nil {
			log.Fatal(err)
		}
		scopes = append(scopes, &ps)
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.PermissionPolicy).Insert(pp.ID, pp, &insertOpt)
	if err != nil {
		return nil, err
	}
	return pp, nil
}

// DeletePermissionPoliciesByPermissionID removes all policy links for a permission.
func (p *provider) DeletePermissionPoliciesByPermissionID(ctx context.Context, permissionID string) error {
	params := make(map[string]interface{}, 1)
	params["permission_id"] = permissionID
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionPolicy)
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

// GetPermissionPolicies returns all policy links for a permission.
func (p *provider) GetPermissionPolicies(ctx context.Context, permissionID string) ([]*schemas.PermissionPolicy, error) {
	policies := []*schemas.PermissionPolicy{}
	params := make(map[string]interface{}, 1)
	params["permission_id"] = permissionID
	query := fmt.Sprintf(`SELECT _id, permission_id, policy_id, created_at FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionPolicy)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var pp schemas.PermissionPolicy
		err := queryResult.Row(&pp)
		if err != nil {
			log.Fatal(err)
		}
		policies = append(policies, &pp)
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return policies, nil
}

// GetPermissionsForResourceScope returns all permissions (with their policies and targets)
// that match a given resource name and scope name. This is the hot-path query used by
// the evaluation engine. Uses sequential queries for clarity.
func (p *provider) GetPermissionsForResourceScope(ctx context.Context, resourceName string, scopeName string) ([]*schemas.PermissionWithPolicies, error) {
	// 1. Find resource by name
	resource, err := p.GetResourceByName(ctx, resourceName)
	if err != nil {
		return nil, err
	}

	// 2. Find scope by name
	scope, err := p.GetScopeByName(ctx, scopeName)
	if err != nil {
		return nil, err
	}

	// 3. Find permissions for this resource
	permParams := make(map[string]interface{}, 1)
	permParams["resource_id"] = resource.ID
	permQuery := fmt.Sprintf(`SELECT _id, name, description, resource_id, decision_strategy, created_at, updated_at FROM %s.%s WHERE resource_id=$resource_id`, p.scopeName, schemas.Collections.Permission)
	permResult, err := p.db.Query(permQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: permParams,
	})
	if err != nil {
		return nil, err
	}
	var permissions []schemas.Permission
	for permResult.Next() {
		var perm schemas.Permission
		if err := permResult.Row(&perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}
	if err := permResult.Err(); err != nil {
		return nil, err
	}

	if len(permissions) == 0 {
		return nil, nil
	}

	// 4. For each permission, check if it has the requested scope
	var result []*schemas.PermissionWithPolicies

	for _, perm := range permissions {
		// Check if this permission has the requested scope
		scopeCheckParams := make(map[string]interface{}, 2)
		scopeCheckParams["permission_id"] = perm.ID
		scopeCheckParams["scope_id"] = scope.ID
		scopeCountQuery := fmt.Sprintf(`SELECT COUNT(*) as Total FROM %s.%s WHERE permission_id=$permission_id AND scope_id=$scope_id`, p.scopeName, schemas.Collections.PermissionScope)
		scopeCountResult, err := p.db.Query(scopeCountQuery, &gocb.QueryOptions{
			Context:         ctx,
			ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
			NamedParameters: scopeCheckParams,
		})
		if err != nil {
			return nil, err
		}
		var countDocs TotalDocs
		err = scopeCountResult.One(&countDocs)
		if err != nil {
			return nil, err
		}
		if countDocs.Total == 0 {
			continue
		}

		// 5. Find permission_policies for this permission
		ppParams := make(map[string]interface{}, 1)
		ppParams["permission_id"] = perm.ID
		ppQuery := fmt.Sprintf(`SELECT _id, permission_id, policy_id, created_at FROM %s.%s WHERE permission_id=$permission_id`, p.scopeName, schemas.Collections.PermissionPolicy)
		ppResult, err := p.db.Query(ppQuery, &gocb.QueryOptions{
			Context:         ctx,
			ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
			NamedParameters: ppParams,
		})
		if err != nil {
			return nil, err
		}
		var permPolicies []schemas.PermissionPolicy
		for ppResult.Next() {
			var pp schemas.PermissionPolicy
			if err := ppResult.Row(&pp); err != nil {
				return nil, err
			}
			permPolicies = append(permPolicies, pp)
		}
		if err := ppResult.Err(); err != nil {
			return nil, err
		}

		if len(permPolicies) == 0 {
			continue
		}

		// 6. For each permission_policy, resolve the policy and its targets
		var policiesWithTargets []schemas.PolicyWithTargets
		for _, pp := range permPolicies {
			policy, err := p.GetPolicyByID(ctx, pp.PolicyID)
			if err != nil {
				return nil, err
			}

			// Get targets for this policy
			targetParams := make(map[string]interface{}, 1)
			targetParams["policy_id"] = policy.ID
			targetQuery := fmt.Sprintf(`SELECT _id, policy_id, target_type, target_value, created_at FROM %s.%s WHERE policy_id=$policy_id`, p.scopeName, schemas.Collections.PolicyTarget)
			targetResult, err := p.db.Query(targetQuery, &gocb.QueryOptions{
				Context:         ctx,
				ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
				NamedParameters: targetParams,
			})
			if err != nil {
				return nil, err
			}
			var targets []schemas.PolicyTargetView
			for targetResult.Next() {
				var target schemas.PolicyTarget
				if err := targetResult.Row(&target); err != nil {
					return nil, err
				}
				targets = append(targets, schemas.PolicyTargetView{
					TargetType:  target.TargetType,
					TargetValue: target.TargetValue,
				})
			}
			if err := targetResult.Err(); err != nil {
				return nil, err
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
