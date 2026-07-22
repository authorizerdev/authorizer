package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimGroup creates a new SCIM group. DisplayName uniqueness within an org
// is enforced by the caller (service layer), not the DB.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.CreatedAt = time.Now().Unix()
	group.UpdatedAt = time.Now().Unix()
	return group, nil
}

// GetScimGroupByID fetches a group by primary key.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	return nil, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName in an org.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	return nil, nil
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId in an org.
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	return nil, nil
}

// UpdateScimGroup writes back a fully-loaded record (PUT displayName change).
// Callers MUST load-then-mutate — Save writes every column.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	group.UpdatedAt = time.Now().Unix()
	return group, nil
}

// DeleteScimGroup removes a group.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	return nil
}
