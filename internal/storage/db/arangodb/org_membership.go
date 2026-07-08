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

// AddOrgMembership creates a new membership. The unique hash index on
// (org_id, user_id) rejects duplicates at the database layer.
func (p *provider) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.ID == "" {
		membership.ID = uuid.New().String()
	}
	membership.Key = membership.ID
	now := time.Now().Unix()
	membership.CreatedAt = now
	membership.UpdatedAt = now
	membershipCollection, _ := p.db.Collection(ctx, schemas.Collections.OrgMembership)
	doc, err := structToDocument(membership)
	if err != nil {
		return nil, err
	}
	meta, err := membershipCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	membership.Key = meta.Key
	membership.ID = meta.ID.String()
	return membership, nil
}

// UpdateOrgMembership updates a membership record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	membership.UpdatedAt = time.Now().Unix()
	membershipCollection, _ := p.db.Collection(ctx, schemas.Collections.OrgMembership)
	doc, err := structToDocument(membership)
	if err != nil {
		return nil, err
	}
	meta, err := membershipCollection.UpdateDocument(ctx, membership.Key, doc)
	if err != nil {
		return nil, err
	}
	membership.Key = meta.Key
	membership.ID = meta.ID.String()
	return membership, nil
}

// DeleteOrgMembership removes a membership record.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	membershipCollection, _ := p.db.Collection(ctx, schemas.Collections.OrgMembership)
	_, err := membershipCollection.RemoveDocument(ctx, membership.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	var membership *schemas.OrgMembership
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id AND d.user_id == @user_id LIMIT 1 RETURN d", schemas.Collections.OrgMembership)
	bindVars := map[string]interface{}{
		"org_id":  orgID,
		"user_id": userID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if membership == nil {
				return nil, fmt.Errorf("org membership not found")
			}
			break
		}
		m := &schemas.OrgMembership{}
		if _, err := readDocument(ctx, cursor, m); err != nil {
			return nil, err
		}
		membership = m
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

func (p *provider) listOrgMemberships(ctx context.Context, filterField, filterValue string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	memberships := []*schemas.OrgMembership{}
	query := fmt.Sprintf("FOR d in %s FILTER d.%s == @filter_value SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.OrgMembership, filterField, pagination.Offset, pagination.Limit)
	bindVars := map[string]interface{}{
		"filter_value": filterValue,
	}
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, bindVars)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		membership := &schemas.OrgMembership{}
		meta, err := readDocument(ctx, cursor, membership)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			memberships = append(memberships, membership)
		}
	}
	return memberships, paginationClone, nil
}
