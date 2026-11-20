package surrealdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimGroup to save a SCIM group in database
func (p *provider) AddScimGroup(ctx context.Context, v *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	v.Key = v.ID
	now := time.Now().Unix()
	v.CreatedAt = now
	v.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.ScimGroup, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateScimGroup to update a SCIM group in database. Callers MUST load the record first:
// every field is written.
func (p *provider) UpdateScimGroup(ctx context.Context, v *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if v.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	v.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.ScimGroup, v.ID, v); err != nil {
		return nil, err
	}
	return v, nil
}

// DeleteScimGroup to delete a SCIM group
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	return p.deleteByID(ctx, schemas.Collections.ScimGroup, group.ID)
}

// GetScimGroupByID to get a SCIM group by ID
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	return getByID[schemas.ScimGroup](ctx, p, schemas.Collections.ScimGroup, id, "scim group")
}

// groupScanSafetyCap bounds the displayName scan below, the same circuit
// breaker the SQL, Cassandra and DynamoDB providers use.
const groupScanSafetyCap = 100_000

// GetScimGroupByOrgAndDisplayName returns an organization's SCIM group by
// display name, compared case-insensitively (SCIM displayName is
// case-insensitive). The comparison is Go's strings.EqualFold, as in the SQL
// provider, so Unicode case folding is identical across backends.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	groups, err := queryAll[schemas.ScimGroup](ctx, p, selectFields+"type::table($tb) WHERE org_id = $o LIMIT $cap;",
		map[string]any{"tb": schemas.Collections.ScimGroup, "o": orgID, "cap": groupScanSafetyCap})
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if strings.EqualFold(groups[i].DisplayName, displayName) {
			return &groups[i], nil
		}
	}
	if len(groups) >= groupScanSafetyCap {
		p.dependencies.Log.Warn().Str("org_id", orgID).Int("examined", len(groups)).
			Msg("GetScimGroupByOrgAndDisplayName: hit the scan safety cap without a match")
	}
	return nil, fmt.Errorf("scim group: %w", ErrNotFound)
}

// GetScimGroupByOrgAndExternalID to get an organization's SCIM group by external ID
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	// externalId is stored org-namespaced ("<orgID>:<raw>"), like
	// User.ExternalID, so this can never resolve another org's group.
	return getWhere[schemas.ScimGroup](ctx, p, schemas.Collections.ScimGroup, "org_id = $o AND external_id = $e", "scim group", map[string]any{"o": orgID, "e": orgID + ":" + externalID})
}
