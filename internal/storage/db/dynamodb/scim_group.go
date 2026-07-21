package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// groupScanSafetyCap bounds the org-scoped scan-and-compare-in-app lookups
// below (queryEqUntil). It is NOT a realistic ceiling on an org's group
// count — normal lookups page through the whole partition — it exists purely
// as a circuit-breaker against unbounded work on a pathologically large one.
const groupScanSafetyCap = 100_000

// AddScimGroup creates a new SCIM group record.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.Key = group.ID
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	if err := p.putItem(ctx, schemas.Collections.ScimGroup, group); err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateScimGroup updates a SCIM group record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge, so a partial struct
// blanks untouched columns to their zero values.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	group.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.ScimGroup, "id", group.ID, group); err != nil {
		return nil, err
	}
	return group, nil
}

// DeleteScimGroup removes a SCIM group record.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	if group == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.ScimGroup, "id", group.ID)
}

// GetScimGroupByID fetches a SCIM group by primary key.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	err := p.getItemByHash(ctx, schemas.Collections.ScimGroup, "id", id, &group)
	if err != nil {
		return nil, err
	}
	if group.ID == "" {
		return nil, errors.New("no document found")
	}
	return &group, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org. There is no GSI on display_name, so query the
// org_id GSI and match displayName in-app (an org's group set is small).
// displayName is compared case-insensitively: SCIM Group.displayName is
// caseExact:false (RFC 7644 §3.4.2.2), and a DynamoDB GSI lookup is exact-match
// only, so the case-fold happens here in Go with strings.EqualFold.
//
// Note for callers relying on this for uniqueness (service.ensureDisplayNameFree
// / CreateGroup dedup): the org_id GSI is EVENTUALLY consistent, so a probe run
// immediately after a sibling create/rename on this org may not observe it yet.
// Combined with the check-then-insert nature of the callers, this is a known,
// accepted race — see their doc comments.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	var found schemas.ScimGroup
	ok, err := p.queryEqUntil(ctx, schemas.Collections.ScimGroup, "org_id", "org_id", orgID, groupScanSafetyCap, func(it map[string]types.AttributeValue) (bool, error) {
		var group schemas.ScimGroup
		if err := unmarshalItem(it, &group); err != nil {
			return false, err
		}
		if !strings.EqualFold(group.DisplayName, displayName) {
			return false, nil
		}
		found = group
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("no document found")
	}
	return &found, nil
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId within an org. externalId is stored org-namespaced ("<orgID>:<raw>")
// exactly like User.ExternalID. No GSI on external_id, so query the org_id GSI
// and match in-app (an org's group set is small).
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	want := orgID + ":" + externalID
	var found schemas.ScimGroup
	ok, err := p.queryEqUntil(ctx, schemas.Collections.ScimGroup, "org_id", "org_id", orgID, groupScanSafetyCap, func(it map[string]types.AttributeValue) (bool, error) {
		var group schemas.ScimGroup
		if err := unmarshalItem(it, &group); err != nil {
			return false, err
		}
		if group.ExternalID == nil || *group.ExternalID != want {
			return false, nil
		}
		found = group
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("no document found")
	}
	return &found, nil
}
