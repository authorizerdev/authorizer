package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const orgMembershipColumns = "_id, org_id, user_id, roles, created_at, updated_at"

// AddOrgMembership creates a new membership. (org_id, user_id) is unique;
// Couchbase has no compound unique constraint, so guard with a
// check-then-insert (closes the sequential case).
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
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(membership)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.OrgMembership).Insert(membership.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// UpdateOrgMembership updates a membership record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	membership.UpdatedAt = time.Now().Unix()
	membershipMap, err := structToDocument(membership)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(membershipMap)
	params["_id"] = membership.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.OrgMembership, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// DeleteOrgMembership removes a membership record.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.OrgMembership).Remove(membership.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	params := make(map[string]interface{}, 2)
	params["org_id"] = orgID
	params["user_id"] = userID
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id AND user_id=$user_id LIMIT 1`, orgMembershipColumns, p.scopeName, schemas.Collections.OrgMembership)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	membership := &schemas.OrgMembership{}
	if err := decodeDocument(raw, membership); err != nil {
		return nil, err
	}
	return membership, nil
}

// ListOrgMembershipsByOrg returns paginated memberships of an organization.
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "org_id", orgID, pagination)
}

// ListOrgMembershipsByUser returns paginated memberships held by a user.
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "user_id", userID, pagination)
}

func (p *provider) listOrgMemberships(ctx context.Context, filterColumn, filterValue string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	memberships := []*schemas.OrgMembership{}
	paginationClone := pagination
	table := fmt.Sprintf("%s.%s", p.scopeName, schemas.Collections.OrgMembership)

	params := make(map[string]interface{}, 3)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	params["filter_value"] = filterValue

	total := TotalDocs{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s WHERE %s=$filter_value", table, filterColumn)
	countRes, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: map[string]interface{}{"filter_value": filterValue},
	})
	if err != nil {
		return nil, nil, err
	}
	_ = countRes.One(&total)
	paginationClone.Total = total.Total

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s=$filter_value ORDER BY created_at DESC OFFSET $offset LIMIT $limit", orgMembershipColumns, table, filterColumn)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, nil, err
		}
		membership := &schemas.OrgMembership{}
		if err := decodeDocument(raw, membership); err != nil {
			return nil, nil, err
		}
		memberships = append(memberships, membership)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return memberships, paginationClone, nil
}
