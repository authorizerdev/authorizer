package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgMembership creates a new membership. The (org_id, user_id) pair is
// unique — adding a duplicate returns an error.
func (p *provider) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.ID == "" {
		membership.ID = uuid.New().String()
	}
	membership.CreatedAt = time.Now().Unix()
	membership.UpdatedAt = time.Now().Unix()
	return membership, nil
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	return nil, nil
}

// UpdateOrgMembership updates the roles of an existing membership.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	membership.UpdatedAt = time.Now().Unix()
	return membership, nil
}

// DeleteOrgMembership removes a membership.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	return nil
}

// ListOrgMembershipsByOrg returns paginated memberships of an organization.
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return nil, nil, nil
}

// ListOrgMembershipsByUser returns paginated memberships held by a user.
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return nil, nil, nil
}
