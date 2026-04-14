package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	collection := p.db.Collection(schemas.Collections.Policy, options.Collection())
	_, err := collection.InsertOne(ctx, policy)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// UpdatePolicy updates an existing authorization policy.
func (p *provider) UpdatePolicy(ctx context.Context, policy *schemas.Policy) (*schemas.Policy, error) {
	policy.UpdatedAt = time.Now().Unix()
	collection := p.db.Collection(schemas.Collections.Policy, options.Collection())
	_, err := collection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": policy.ID}}, bson.M{"$set": policy}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// DeletePolicy deletes an authorization policy by ID.
// Returns an error if any permission_policy references this policy.
// Cascade-deletes associated policy targets.
func (p *provider) DeletePolicy(ctx context.Context, id string) error {
	permissionPolicyCollection := p.db.Collection(schemas.Collections.PermissionPolicy, options.Collection())
	count, err := permissionPolicyCollection.CountDocuments(ctx, bson.M{"policy_id": id}, options.Count())
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete policy: %d permission_policy(s) reference it", count)
	}
	// Cascade-delete policy targets
	policyTargetCollection := p.db.Collection(schemas.Collections.PolicyTarget, options.Collection())
	_, err = policyTargetCollection.DeleteMany(ctx, bson.M{"policy_id": id}, options.Delete())
	if err != nil {
		return err
	}
	collection := p.db.Collection(schemas.Collections.Policy, options.Collection())
	_, err = collection.DeleteOne(ctx, bson.M{"_id": id}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyByID returns an authorization policy by its ID.
func (p *provider) GetPolicyByID(ctx context.Context, id string) (*schemas.Policy, error) {
	var policy schemas.Policy
	collection := p.db.Collection(schemas.Collections.Policy, options.Collection())
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&policy)
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// ListPolicies returns a paginated list of authorization policies.
func (p *provider) ListPolicies(ctx context.Context, pagination *model.Pagination) ([]*schemas.Policy, *model.Pagination, error) {
	policies := []*schemas.Policy{}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	collection := p.db.Collection(schemas.Collections.Policy, options.Collection())
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
		var policy *schemas.Policy
		err := cursor.Decode(&policy)
		if err != nil {
			return nil, nil, err
		}
		policies = append(policies, policy)
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
	collection := p.db.Collection(schemas.Collections.PolicyTarget, options.Collection())
	_, err := collection.InsertOne(ctx, target)
	if err != nil {
		return nil, err
	}
	return target, nil
}

// DeletePolicyTargetsByPolicyID removes all targets for a policy.
func (p *provider) DeletePolicyTargetsByPolicyID(ctx context.Context, policyID string) error {
	collection := p.db.Collection(schemas.Collections.PolicyTarget, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"policy_id": policyID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyTargets returns all targets for a policy.
func (p *provider) GetPolicyTargets(ctx context.Context, policyID string) ([]*schemas.PolicyTarget, error) {
	targets := []*schemas.PolicyTarget{}
	collection := p.db.Collection(schemas.Collections.PolicyTarget, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{"policy_id": policyID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var target *schemas.PolicyTarget
		err := cursor.Decode(&target)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}
