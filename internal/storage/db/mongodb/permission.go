package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	collection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	_, err := collection.InsertOne(ctx, permission)
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// UpdatePermission updates an existing authorization permission.
func (p *provider) UpdatePermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	permission.UpdatedAt = time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	_, err := collection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": permission.ID}}, bson.M{"$set": permission}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return permission, nil
}

// DeletePermission deletes an authorization permission by ID.
// Cascade-deletes associated permission_scopes and permission_policies.
func (p *provider) DeletePermission(ctx context.Context, id string) error {
	permissionScopeCollection := p.db.Collection(schemas.Collections.PermissionScope, options.Collection())
	_, err := permissionScopeCollection.DeleteMany(ctx, bson.M{"permission_id": id}, options.Delete())
	if err != nil {
		return err
	}
	permissionPolicyCollection := p.db.Collection(schemas.Collections.PermissionPolicy, options.Collection())
	_, err = permissionPolicyCollection.DeleteMany(ctx, bson.M{"permission_id": id}, options.Delete())
	if err != nil {
		return err
	}
	collection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	_, err = collection.DeleteOne(ctx, bson.M{"_id": id}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetPermissionByID returns an authorization permission by its ID.
func (p *provider) GetPermissionByID(ctx context.Context, id string) (*schemas.Permission, error) {
	var permission schemas.Permission
	collection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&permission)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// ListPermissions returns a paginated list of authorization permissions.
func (p *provider) ListPermissions(ctx context.Context, pagination *model.Pagination) ([]*schemas.Permission, *model.Pagination, error) {
	permissions := []*schemas.Permission{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	collection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	count, err := collection.CountDocuments(ctx, bson.M{}, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var permission *schemas.Permission
		err := cursor.Decode(&permission)
		if err != nil {
			return nil, nil, err
		}
		permissions = append(permissions, permission)
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
	collection := p.db.Collection(schemas.Collections.PermissionScope, options.Collection())
	_, err := collection.InsertOne(ctx, ps)
	if err != nil {
		return nil, err
	}
	return ps, nil
}

// DeletePermissionScopesByPermissionID removes all scope links for a permission.
func (p *provider) DeletePermissionScopesByPermissionID(ctx context.Context, permissionID string) error {
	collection := p.db.Collection(schemas.Collections.PermissionScope, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"permission_id": permissionID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetPermissionScopes returns all scope links for a permission.
func (p *provider) GetPermissionScopes(ctx context.Context, permissionID string) ([]*schemas.PermissionScope, error) {
	scopes := []*schemas.PermissionScope{}
	collection := p.db.Collection(schemas.Collections.PermissionScope, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{"permission_id": permissionID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var ps *schemas.PermissionScope
		err := cursor.Decode(&ps)
		if err != nil {
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
	collection := p.db.Collection(schemas.Collections.PermissionPolicy, options.Collection())
	_, err := collection.InsertOne(ctx, pp)
	if err != nil {
		return nil, err
	}
	return pp, nil
}

// DeletePermissionPoliciesByPermissionID removes all policy links for a permission.
func (p *provider) DeletePermissionPoliciesByPermissionID(ctx context.Context, permissionID string) error {
	collection := p.db.Collection(schemas.Collections.PermissionPolicy, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"permission_id": permissionID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetPermissionPolicies returns all policy links for a permission.
func (p *provider) GetPermissionPolicies(ctx context.Context, permissionID string) ([]*schemas.PermissionPolicy, error) {
	policies := []*schemas.PermissionPolicy{}
	collection := p.db.Collection(schemas.Collections.PermissionPolicy, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{"permission_id": permissionID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var pp *schemas.PermissionPolicy
		err := cursor.Decode(&pp)
		if err != nil {
			return nil, err
		}
		policies = append(policies, pp)
	}
	return policies, nil
}

// GetPermissionsForResourceScope returns all permissions (with their policies and targets)
// that match a given resource name and scope name. This is the hot-path query used by
// the evaluation engine. Uses sequential queries for clarity.
func (p *provider) GetPermissionsForResourceScope(ctx context.Context, resourceName string, scopeName string) ([]*schemas.PermissionWithPolicies, error) {
	// 1. Find resource by name
	var resource schemas.Resource
	resourceCollection := p.db.Collection(schemas.Collections.Resource, options.Collection())
	err := resourceCollection.FindOne(ctx, bson.M{"name": resourceName}).Decode(&resource)
	if err != nil {
		return nil, err
	}

	// 2. Find scope by name
	var scope schemas.Scope
	scopeCollection := p.db.Collection(schemas.Collections.Scope, options.Collection())
	err = scopeCollection.FindOne(ctx, bson.M{"name": scopeName}).Decode(&scope)
	if err != nil {
		return nil, err
	}

	// 3. Find permissions for this resource
	permissionCollection := p.db.Collection(schemas.Collections.Permission, options.Collection())
	permCursor, err := permissionCollection.Find(ctx, bson.M{"resource_id": resource.ID})
	if err != nil {
		return nil, err
	}
	defer permCursor.Close(ctx)

	var permissions []schemas.Permission
	for permCursor.Next(ctx) {
		var perm schemas.Permission
		if err := permCursor.Decode(&perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	if len(permissions) == 0 {
		return nil, nil
	}

	// 4. For each permission, check if it has the requested scope
	permissionScopeCollection := p.db.Collection(schemas.Collections.PermissionScope, options.Collection())
	permissionPolicyCollection := p.db.Collection(schemas.Collections.PermissionPolicy, options.Collection())
	policyCollection := p.db.Collection(schemas.Collections.Policy, options.Collection())
	policyTargetCollection := p.db.Collection(schemas.Collections.PolicyTarget, options.Collection())

	var result []*schemas.PermissionWithPolicies

	for _, perm := range permissions {
		// Check if this permission has the requested scope
		scopeCount, err := permissionScopeCollection.CountDocuments(ctx, bson.M{
			"permission_id": perm.ID,
			"scope_id":      scope.ID,
		}, options.Count())
		if err != nil {
			return nil, err
		}
		if scopeCount == 0 {
			continue
		}

		// 5. Find permission_policies for this permission
		ppCursor, err := permissionPolicyCollection.Find(ctx, bson.M{"permission_id": perm.ID})
		if err != nil {
			return nil, err
		}

		var permPolicies []schemas.PermissionPolicy
		for ppCursor.Next(ctx) {
			var pp schemas.PermissionPolicy
			if err := ppCursor.Decode(&pp); err != nil {
				ppCursor.Close(ctx)
				return nil, err
			}
			permPolicies = append(permPolicies, pp)
		}
		ppCursor.Close(ctx)

		if len(permPolicies) == 0 {
			continue
		}

		// 6. For each permission_policy, resolve the policy and its targets
		var policiesWithTargets []schemas.PolicyWithTargets
		for _, pp := range permPolicies {
			var policy schemas.Policy
			err := policyCollection.FindOne(ctx, bson.M{"_id": pp.PolicyID}).Decode(&policy)
			if err != nil {
				return nil, err
			}

			// Get targets for this policy
			targetCursor, err := policyTargetCollection.Find(ctx, bson.M{"policy_id": policy.ID})
			if err != nil {
				return nil, err
			}

			var targets []schemas.PolicyTargetView
			for targetCursor.Next(ctx) {
				var target schemas.PolicyTarget
				if err := targetCursor.Decode(&target); err != nil {
					targetCursor.Close(ctx)
					return nil, err
				}
				targets = append(targets, schemas.PolicyTargetView{
					TargetType:  target.TargetType,
					TargetValue: target.TargetValue,
				})
			}
			targetCursor.Close(ctx)

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
