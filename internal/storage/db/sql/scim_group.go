package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimGroup creates a new SCIM group record.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.Key = group.ID
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	res := p.db.Create(group)
	if res.Error != nil {
		return nil, res.Error
	}
	return group, nil
}

// UpdateScimGroup updates a SCIM group record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	group.UpdatedAt = time.Now().Unix()
	res := p.db.Save(group)
	if res.Error != nil {
		return nil, res.Error
	}
	return group, nil
}

// DeleteScimGroup removes a SCIM group.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	return p.db.Delete(group).Error
}

// GetScimGroupByID fetches a SCIM group by primary key.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	res := p.db.Where("id = ?", id).First(&group)
	if res.Error != nil {
		return nil, res.Error
	}
	return &group, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	res := p.db.Where("org_id = ? AND display_name = ?", orgID, displayName).First(&group)
	if res.Error != nil {
		return nil, res.Error
	}
	return &group, nil
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId within an org. externalId is stored org-namespaced ("<orgID>:<raw>")
// exactly like User.ExternalID, so this can never resolve another org's group.
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	res := p.db.Where("org_id = ? AND external_id = ?", orgID, orgID+":"+externalID).First(&group)
	if res.Error != nil {
		return nil, res.Error
	}
	return &group, nil
}
