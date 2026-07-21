package sql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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

// groupScanSafetyCap bounds the org-scoped scan-and-compare-in-app
// displayName lookup below. It is NOT a realistic ceiling on an org's group
// count, purely a circuit-breaker against unbounded work on a pathologically
// large org, mirroring the equivalent cap in the Cassandra/DynamoDB providers.
const groupScanSafetyCap = 100_000

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org. displayName is compared case-insensitively:
// SCIM Group.displayName is caseExact:false (RFC 7644 §3.4.2.2).
//
// This deliberately does NOT use SQL's LOWER() for the comparison: it isn't
// actually portable the way it looks. SQLite's built-in LOWER() folds ASCII
// only, while Postgres/MySQL/SQL Server's LOWER() are collation/Unicode-aware
// in ways that differ from each other AND from Go's strings.ToLower — so a
// non-ASCII displayName (e.g. "CAFÉ") could match on one dialect and miss on
// another, and none of them are guaranteed to agree with the Go-side fold
// every other provider (Cassandra, DynamoDB, Mongo's collation) uses. Fetching
// by org (indexed, cheap) and comparing with strings.EqualFold in Go instead
// makes all 6 storage backends fold identically, matching the Cassandra/
// DynamoDB fetch-then-filter shape exactly.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	var groups []schemas.ScimGroup
	res := p.db.Where("org_id = ?", orgID).Limit(groupScanSafetyCap).Find(&groups)
	if res.Error != nil {
		return nil, res.Error
	}
	for _, g := range groups {
		if strings.EqualFold(g.DisplayName, displayName) {
			return &g, nil
		}
	}
	if len(groups) >= groupScanSafetyCap {
		p.dependencies.Log.Warn().Str("org_id", orgID).Int("examined", len(groups)).
			Msg("GetScimGroupByOrgAndDisplayName: hit the scan safety cap without a match")
	}
	return nil, gorm.ErrRecordNotFound
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
