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

// AddPermission creates a new authorization permission.
func (p *provider) AddPermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}
	permission.Key = permission.ID
	permission.CreatedAt = time.Now().Unix()
	permission.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.Permission)
	meta, err := collection.CreateDocument(ctx, permission)
	if err != nil {
		return nil, err
	}
	permission.Key = meta.Key
	permission.ID = meta.ID.String()
	return permission, nil
}

// UpdatePermission updates an existing authorization permission.
func (p *provider) UpdatePermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	permission.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, schemas.Collections.Permission)
	meta, err := collection.UpdateDocument(ctx, permission.Key, permission)
	if err != nil {
		return nil, err
	}
	permission.Key = meta.Key
	permission.ID = meta.ID.String()
	return permission, nil
}

// DeletePermission deletes an authorization permission by ID.
// Cascade-deletes associated permission_scopes and permission_policies.
func (p *provider) DeletePermission(ctx context.Context, id string) error {
	// Cascade-delete permission_scopes
	deleteScopesQuery := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id REMOVE d IN %s", schemas.Collections.PermissionScope, schemas.Collections.PermissionScope)
	scopeCursor, err := p.db.Query(ctx, deleteScopesQuery, map[string]interface{}{
		"permission_id": id,
	})
	if err != nil {
		return err
	}
	defer scopeCursor.Close()

	// Cascade-delete permission_policies
	deletePoliciesQuery := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id REMOVE d IN %s", schemas.Collections.PermissionPolicy, schemas.Collections.PermissionPolicy)
	policyCursor, err := p.db.Query(ctx, deletePoliciesQuery, map[string]interface{}{
		"permission_id": id,
	})
	if err != nil {
		return err
	}
	defer policyCursor.Close()

	// Find the document key for this permission
	permission, err := p.GetPermissionByID(ctx, id)
	if err != nil {
		return err
	}
	collection, _ := p.db.Collection(ctx, schemas.Collections.Permission)
	_, err = collection.RemoveDocument(ctx, permission.Key)
	return err
}

// GetPermissionByID returns an authorization permission by its ID.
func (p *provider) GetPermissionByID(ctx context.Context, id string) (*schemas.Permission, error) {
	var permission *schemas.Permission
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id RETURN d", schemas.Collections.Permission)
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
			if permission == nil {
				return nil, fmt.Errorf("permission not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &permission)
		if err != nil {
			return nil, err
		}
	}
	return permission, nil
}

// ListPermissions returns a paginated list of authorization permissions.
func (p *provider) ListPermissions(ctx context.Context, pagination *model.Pagination) ([]*schemas.Permission, *model.Pagination, error) {
	permissions := []*schemas.Permission{}
	query := fmt.Sprintf("FOR d IN %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Permission, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var permission *schemas.Permission
		meta, err := cursor.ReadDocument(ctx, &permission)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			permissions = append(permissions, permission)
		}
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
	collection, _ := p.db.Collection(ctx, schemas.Collections.PermissionScope)
	meta, err := collection.CreateDocument(ctx, ps)
	if err != nil {
		return nil, err
	}
	ps.Key = meta.Key
	ps.ID = meta.ID.String()
	return ps, nil
}

// DeletePermissionScopesByPermissionID removes all scope links for a permission.
func (p *provider) DeletePermissionScopesByPermissionID(ctx context.Context, permissionID string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id REMOVE d IN %s", schemas.Collections.PermissionScope, schemas.Collections.PermissionScope)
	cursor, err := p.db.Query(ctx, query, map[string]interface{}{
		"permission_id": permissionID,
	})
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetPermissionScopes returns all scope links for a permission.
func (p *provider) GetPermissionScopes(ctx context.Context, permissionID string) ([]*schemas.PermissionScope, error) {
	scopes := []*schemas.PermissionScope{}
	query := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id RETURN d", schemas.Collections.PermissionScope)
	cursor, err := p.db.Query(ctx, query, map[string]interface{}{
		"permission_id": permissionID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		var ps *schemas.PermissionScope
		if _, err := cursor.ReadDocument(ctx, &ps); arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		scopes = append(scopes, ps)
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
	collection, _ := p.db.Collection(ctx, schemas.Collections.PermissionPolicy)
	meta, err := collection.CreateDocument(ctx, pp)
	if err != nil {
		return nil, err
	}
	pp.Key = meta.Key
	pp.ID = meta.ID.String()
	return pp, nil
}

// DeletePermissionPoliciesByPermissionID removes all policy links for a permission.
func (p *provider) DeletePermissionPoliciesByPermissionID(ctx context.Context, permissionID string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id REMOVE d IN %s", schemas.Collections.PermissionPolicy, schemas.Collections.PermissionPolicy)
	cursor, err := p.db.Query(ctx, query, map[string]interface{}{
		"permission_id": permissionID,
	})
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetPermissionPolicies returns all policy links for a permission.
func (p *provider) GetPermissionPolicies(ctx context.Context, permissionID string) ([]*schemas.PermissionPolicy, error) {
	policies := []*schemas.PermissionPolicy{}
	query := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id RETURN d", schemas.Collections.PermissionPolicy)
	cursor, err := p.db.Query(ctx, query, map[string]interface{}{
		"permission_id": permissionID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		var pp *schemas.PermissionPolicy
		if _, err := cursor.ReadDocument(ctx, &pp); arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		policies = append(policies, pp)
	}
	return policies, nil
}

// GetPermissionsForResourceScope returns all permissions (with their policies and targets)
// that match a given resource name and scope name. This is the hot-path query used by
// the evaluation engine. Uses sequential lookups across collections.
func (p *provider) GetPermissionsForResourceScope(ctx context.Context, resourceName string, scopeName string) ([]*schemas.PermissionWithPolicies, error) {
	// Step 1: Find the resource by name
	resource, err := p.GetResourceByName(ctx, resourceName)
	if err != nil {
		return nil, nil // Resource not found means no permissions
	}

	// Step 2: Find the scope by name
	scope, err := p.GetScopeByName(ctx, scopeName)
	if err != nil {
		return nil, nil // Scope not found means no permissions
	}

	// Step 3: Find permissions for this resource
	permQuery := fmt.Sprintf("FOR d IN %s FILTER d.resource_id == @resource_id RETURN d", schemas.Collections.Permission)
	permCursor, err := p.db.Query(ctx, permQuery, map[string]interface{}{
		"resource_id": resource.ID,
	})
	if err != nil {
		return nil, err
	}
	defer permCursor.Close()

	var permissions []*schemas.Permission
	for {
		var perm *schemas.Permission
		if _, err := permCursor.ReadDocument(ctx, &perm); arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	if len(permissions) == 0 {
		return nil, nil
	}

	// Step 4: Filter permissions that have this scope linked
	var matchedPermissions []*schemas.Permission
	for _, perm := range permissions {
		psQuery := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id AND d.scope_id == @scope_id RETURN d", schemas.Collections.PermissionScope)
		psCursor, err := p.db.Query(ctx, psQuery, map[string]interface{}{
			"permission_id": perm.ID,
			"scope_id":      scope.ID,
		})
		if err != nil {
			return nil, err
		}
		var ps *schemas.PermissionScope
		if _, err := psCursor.ReadDocument(ctx, &ps); err == nil && ps != nil {
			matchedPermissions = append(matchedPermissions, perm)
		}
		psCursor.Close()
	}

	if len(matchedPermissions) == 0 {
		return nil, nil
	}

	// Step 5: For each matched permission, resolve policies and targets
	var result []*schemas.PermissionWithPolicies
	for _, perm := range matchedPermissions {
		pwp := &schemas.PermissionWithPolicies{
			PermissionID:     perm.ID,
			PermissionName:   perm.Name,
			DecisionStrategy: perm.DecisionStrategy,
		}

		// Get permission_policies for this permission
		ppQuery := fmt.Sprintf("FOR d IN %s FILTER d.permission_id == @permission_id RETURN d", schemas.Collections.PermissionPolicy)
		ppCursor, err := p.db.Query(ctx, ppQuery, map[string]interface{}{
			"permission_id": perm.ID,
		})
		if err != nil {
			return nil, err
		}

		var permPolicies []*schemas.PermissionPolicy
		for {
			var pp *schemas.PermissionPolicy
			if _, err := ppCursor.ReadDocument(ctx, &pp); arangoDriver.IsNoMoreDocuments(err) {
				break
			} else if err != nil {
				ppCursor.Close()
				return nil, err
			}
			permPolicies = append(permPolicies, pp)
		}
		ppCursor.Close()

		// For each linked policy, resolve the policy and its targets
		for _, pp := range permPolicies {
			policy, err := p.GetPolicyByID(ctx, pp.PolicyID)
			if err != nil {
				continue // Skip policies that can't be found
			}

			pwt := schemas.PolicyWithTargets{
				PolicyID:         policy.ID,
				PolicyName:       policy.Name,
				Type:             policy.Type,
				Logic:            policy.Logic,
				DecisionStrategy: policy.DecisionStrategy,
			}

			// Get targets for this policy
			targets, err := p.GetPolicyTargets(ctx, policy.ID)
			if err != nil {
				return nil, err
			}
			for _, t := range targets {
				pwt.Targets = append(pwt.Targets, schemas.PolicyTargetView{
					TargetType:  t.TargetType,
					TargetValue: t.TargetValue,
				})
			}

			pwp.Policies = append(pwp.Policies, pwt)
		}

		result = append(result, pwp)
	}

	return result, nil
}
