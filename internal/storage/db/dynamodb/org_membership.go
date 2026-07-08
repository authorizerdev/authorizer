package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgMembership creates a new membership. (org_id, user_id) is unique;
// DynamoDB has no compound unique constraint, so guard with a check-then-insert
// (closes the sequential case).
func (p *provider) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.ID == "" {
		membership.ID = uuid.New().String()
	}
	membership.Key = membership.ID
	now := time.Now().Unix()
	membership.CreatedAt = now
	membership.UpdatedAt = now
	if existing, _ := p.GetOrgMembership(ctx, membership.OrgID, membership.UserID); existing != nil {
		return nil, fmt.Errorf("membership for org_id %s and user_id %s already exists", membership.OrgID, membership.UserID)
	}
	if err := p.putItem(ctx, schemas.Collections.OrgMembership, membership); err != nil {
		return nil, err
	}
	return membership, nil
}

// UpdateOrgMembership updates a membership record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge, so a partial struct
// blanks untouched columns to their zero values.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	membership.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.OrgMembership, "id", membership.ID, membership); err != nil {
		return nil, err
	}
	return membership, nil
}

// DeleteOrgMembership removes a membership record.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	if membership == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.OrgMembership, "id", membership.ID)
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair via the
// org_id GSI with a user_id filter.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	f := expression.Name("user_id").Equal(expression.Value(userID))
	// Must NOT pass a Limit alongside a FilterExpression: DynamoDB applies Limit
	// BEFORE the filter, so Limit=1 reads one arbitrary item from the org_id
	// partition and returns zero matches for a multi-member org even when the
	// (org_id, user_id) row exists — silently allowing duplicate memberships and
	// breaking member removal. queryEq paginates and filters server-side across
	// the whole partition; the unique (org_id, user_id) invariant means at most one match.
	items, err := p.queryEq(ctx, schemas.Collections.OrgMembership, "org_id", "org_id", orgID, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var membership schemas.OrgMembership
	if err := unmarshalItem(items[0], &membership); err != nil {
		return nil, err
	}
	return &membership, nil
}

// ListOrgMembershipsByOrg returns paginated memberships of an organization.
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "org_id", orgID, pagination)
}

// ListOrgMembershipsByUser returns paginated memberships held by a user.
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "user_id", userID, pagination)
}

func (p *provider) listOrgMemberships(ctx context.Context, indexAndAttr, value string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	paginationClone := pagination
	var memberships []*schemas.OrgMembership

	var items []map[string]types.AttributeValue
	items, err := p.queryEq(ctx, schemas.Collections.OrgMembership, indexAndAttr, indexAndAttr, value, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, it := range items {
		var membership schemas.OrgMembership
		if err := unmarshalItem(it, &membership); err != nil {
			return nil, nil, err
		}
		memberships = append(memberships, &membership)
	}

	sort.Slice(memberships, func(i, j int) bool { return memberships[i].CreatedAt > memberships[j].CreatedAt })
	paginationClone.Total = int64(len(memberships))

	start := int(pagination.Offset)
	if start >= len(memberships) {
		return []*schemas.OrgMembership{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(memberships) {
		end = len(memberships)
	}
	return memberships[start:end], paginationClone, nil
}
