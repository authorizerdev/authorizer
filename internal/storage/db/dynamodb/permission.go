package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	if err := p.putItem(ctx, schemas.Collections.Permission, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// UpdatePermission updates an existing authorization permission.
func (p *provider) UpdatePermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	permission.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Permission, "id", permission.ID, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// DeletePermission deletes an authorization permission by ID.
// Cascade-deletes all permission_scopes and permission_policies for this permission.
func (p *provider) DeletePermission(ctx context.Context, id string) error {
	if err := p.DeletePermissionScopesByPermissionID(ctx, id); err != nil {
		return err
	}
	if err := p.DeletePermissionPoliciesByPermissionID(ctx, id); err != nil {
		return err
	}
	return p.deleteItemByHash(ctx, schemas.Collections.Permission, "id", id)
}

// GetPermissionByID returns an authorization permission by its ID.
func (p *provider) GetPermissionByID(ctx context.Context, id string) (*schemas.Permission, error) {
	var permission schemas.Permission
	if err := p.getItemByHash(ctx, schemas.Collections.Permission, "id", id, &permission); err != nil {
		return nil, err
	}
	if permission.ID == "" {
		return nil, errors.New("no document found")
	}
	return &permission, nil
}

// ListPermissions returns a paginated list of authorization permissions.
func (p *provider) ListPermissions(ctx context.Context, pagination *model.Pagination) ([]*schemas.Permission, *model.Pagination, error) {
	var lastKey map[string]types.AttributeValue
	var iteration int64
	paginationClone := pagination
	var permissions []*schemas.Permission

	count, err := p.scanCount(ctx, schemas.Collections.Permission, nil)
	if err != nil {
		return nil, nil, err
	}

	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		items, next, err := p.scanPageIter(ctx, schemas.Collections.Permission, nil, int32(paginationClone.Limit), lastKey)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			var perm schemas.Permission
			if err := unmarshalItem(it, &perm); err != nil {
				return nil, nil, err
			}
			if paginationClone.Offset == iteration {
				permissions = append(permissions, &perm)
			}
		}
		lastKey = next
		iteration += paginationClone.Limit
		if lastKey == nil {
			break
		}
	}
	paginationClone.Total = count
	return permissions, paginationClone, nil
}

// AddPermissionScope links a scope to a permission.
func (p *provider) AddPermissionScope(ctx context.Context, ps *schemas.PermissionScope) (*schemas.PermissionScope, error) {
	if ps.ID == "" {
		ps.ID = uuid.New().String()
	}
	ps.Key = ps.ID
	ps.CreatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.PermissionScope, ps); err != nil {
		return nil, err
	}
	return ps, nil
}

// DeletePermissionScopesByPermissionID removes all scope links for a permission.
func (p *provider) DeletePermissionScopesByPermissionID(ctx context.Context, permissionID string) error {
	f := expression.Name("permission_id").Equal(expression.Value(permissionID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionScope, nil, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var ps schemas.PermissionScope
		if err := unmarshalItem(it, &ps); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.PermissionScope, "id", ps.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetPermissionScopes returns all scope links for a permission.
func (p *provider) GetPermissionScopes(ctx context.Context, permissionID string) ([]*schemas.PermissionScope, error) {
	f := expression.Name("permission_id").Equal(expression.Value(permissionID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionScope, nil, &f)
	if err != nil {
		return nil, err
	}
	var scopes []*schemas.PermissionScope
	for _, it := range items {
		var ps schemas.PermissionScope
		if err := unmarshalItem(it, &ps); err != nil {
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
	if err := p.putItem(ctx, schemas.Collections.PermissionPolicy, pp); err != nil {
		return nil, err
	}
	return pp, nil
}

// DeletePermissionPoliciesByPermissionID removes all policy links for a permission.
func (p *provider) DeletePermissionPoliciesByPermissionID(ctx context.Context, permissionID string) error {
	f := expression.Name("permission_id").Equal(expression.Value(permissionID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionPolicy, nil, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var pp schemas.PermissionPolicy
		if err := unmarshalItem(it, &pp); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.PermissionPolicy, "id", pp.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetPermissionPolicies returns all policy links for a permission.
func (p *provider) GetPermissionPolicies(ctx context.Context, permissionID string) ([]*schemas.PermissionPolicy, error) {
	f := expression.Name("permission_id").Equal(expression.Value(permissionID))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionPolicy, nil, &f)
	if err != nil {
		return nil, err
	}
	var policies []*schemas.PermissionPolicy
	for _, it := range items {
		var pp schemas.PermissionPolicy
		if err := unmarshalItem(it, &pp); err != nil {
			return nil, err
		}
		policies = append(policies, &pp)
	}
	return policies, nil
}

// GetPermissionsForResourceScope returns all permissions (with their policies and targets)
// that apply to a given resource name and scope name. Used by the evaluation engine.
func (p *provider) GetPermissionsForResourceScope(ctx context.Context, resourceName string, scopeName string) ([]*schemas.PermissionWithPolicies, error) {
	// 1. Find resource by name
	resourceItems, err := p.queryEqLimit(ctx, schemas.Collections.Resource, "name", "name", resourceName, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(resourceItems) == 0 {
		return nil, errors.New("no document found")
	}
	var resource schemas.Resource
	if err := unmarshalItem(resourceItems[0], &resource); err != nil {
		return nil, err
	}

	// 2. Find scope by name
	scopeItems, err := p.queryEqLimit(ctx, schemas.Collections.Scope, "name", "name", scopeName, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(scopeItems) == 0 {
		return nil, errors.New("no document found")
	}
	var scope schemas.Scope
	if err := unmarshalItem(scopeItems[0], &scope); err != nil {
		return nil, err
	}

	// 3. Find permissions for this resource
	f := expression.Name("resource_id").Equal(expression.Value(resource.ID))
	permItems, err := p.scanFilteredAll(ctx, schemas.Collections.Permission, nil, &f)
	if err != nil {
		return nil, err
	}
	if len(permItems) == 0 {
		return nil, nil
	}

	var result []*schemas.PermissionWithPolicies

	for _, permItem := range permItems {
		var perm schemas.Permission
		if err := unmarshalItem(permItem, &perm); err != nil {
			return nil, err
		}

		// 4. Check if this permission has the requested scope
		psFilter := expression.Name("permission_id").Equal(expression.Value(perm.ID)).
			And(expression.Name("scope_id").Equal(expression.Value(scope.ID)))
		psItems, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionScope, nil, &psFilter)
		if err != nil {
			return nil, err
		}
		if len(psItems) == 0 {
			continue
		}

		// 5. Find permission_policies for this permission
		ppFilter := expression.Name("permission_id").Equal(expression.Value(perm.ID))
		ppItems, err := p.scanFilteredAll(ctx, schemas.Collections.PermissionPolicy, nil, &ppFilter)
		if err != nil {
			return nil, err
		}
		if len(ppItems) == 0 {
			continue
		}

		// 6. For each permission_policy, resolve the policy and its targets
		var policiesWithTargets []schemas.PolicyWithTargets
		for _, ppItem := range ppItems {
			var pp schemas.PermissionPolicy
			if err := unmarshalItem(ppItem, &pp); err != nil {
				return nil, err
			}

			var policy schemas.Policy
			if err := p.getItemByHash(ctx, schemas.Collections.Policy, "id", pp.PolicyID, &policy); err != nil {
				return nil, err
			}

			// Get targets for this policy
			tFilter := expression.Name("policy_id").Equal(expression.Value(policy.ID))
			tItems, err := p.scanFilteredAll(ctx, schemas.Collections.PolicyTarget, nil, &tFilter)
			if err != nil {
				return nil, err
			}

			var targets []schemas.PolicyTargetView
			for _, tItem := range tItems {
				var target schemas.PolicyTarget
				if err := unmarshalItem(tItem, &target); err != nil {
					return nil, err
				}
				targets = append(targets, schemas.PolicyTargetView{
					TargetType:  target.TargetType,
					TargetValue: target.TargetValue,
				})
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
