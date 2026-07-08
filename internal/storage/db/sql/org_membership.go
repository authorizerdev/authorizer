package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgMembership creates a new membership. The composite unique index on
// (org_id, user_id) rejects duplicates at the database layer.
func (p *provider) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.ID == "" {
		membership.ID = uuid.New().String()
	}
	membership.Key = membership.ID
	now := time.Now().Unix()
	membership.CreatedAt = now
	membership.UpdatedAt = now
	res := p.db.Create(membership)
	if res.Error != nil {
		return nil, res.Error
	}
	return membership, nil
}

// UpdateOrgMembership updates a membership record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	membership.UpdatedAt = time.Now().Unix()
	res := p.db.Save(membership)
	if res.Error != nil {
		return nil, res.Error
	}
	return membership, nil
}

// DeleteOrgMembership removes a membership record.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	return p.db.Delete(membership).Error
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	var membership schemas.OrgMembership
	res := p.db.Where("org_id = ? AND user_id = ?", orgID, userID).First(&membership)
	if res.Error != nil {
		return nil, res.Error
	}
	return &membership, nil
}

// ListOrgMembershipsByOrg returns paginated memberships of an organization.
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "org_id = ?", orgID, pagination)
}

// ListOrgMembershipsByUser returns paginated memberships held by a user.
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "user_id = ?", userID, pagination)
}

func (p *provider) listOrgMemberships(ctx context.Context, whereClause, whereValue string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	var memberships []*schemas.OrgMembership
	res := p.db.Where(whereClause, whereValue).Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&memberships)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	countRes := p.db.Model(&schemas.OrgMembership{}).Where(whereClause, whereValue).Count(&total)
	if countRes.Error != nil {
		return nil, nil, countRes.Error
	}
	return memberships, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}
