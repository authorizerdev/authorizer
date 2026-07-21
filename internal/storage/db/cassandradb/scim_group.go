package cassandradb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const scimGroupColumns = "id, org_id, display_name, external_id, created_at, updated_at"

// groupScanSafetyCap bounds the org-scoped scan-and-compare-in-app
// displayName lookup below. It is NOT a realistic ceiling on an org's group
// count — gocql's Scanner already pages through the full ALLOW FILTERING
// result set as Next() is called, so a normal lookup runs to full exhaustion
// (or an early match) regardless of how many pages that takes. This exists
// purely as a circuit-breaker against unbounded work on a pathologically
// large partition, mirroring DynamoDB's groupScanSafetyCap.
const groupScanSafetyCap = 100_000

// cassandraGroupPageSize bounds each fetch of the org-scoped displayName scan
// to a real page, so an early match doesn't require pulling a large org's
// entire group set into memory first (see GetScimGroupByOrgAndDisplayName).
const cassandraGroupPageSize = 1000

// scanScimGroup maps the scimGroupColumns projection onto a struct.
func scanScimGroup(scan func(...interface{}) error, group *schemas.ScimGroup) error {
	return scan(&group.ID, &group.OrgID, &group.DisplayName, &group.ExternalID, &group.CreatedAt, &group.UpdatedAt)
}

// AddScimGroup creates a new SCIM group record.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.Key = group.ID
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.ScimGroup, scimGroupColumns)
	err := p.db.Query(insertQuery, group.ID, group.OrgID, group.DisplayName, group.ExternalID, group.CreatedAt, group.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateScimGroup updates a SCIM group record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	group.UpdatedAt = time.Now().Unix()
	groupMap := buildCQLColumnMap(group)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range groupMap {
		if key == "id" || key == "_key" {
			continue
		}
		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}
		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	updateValues = append(updateValues, group.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.ScimGroup, updateFields)
	if err := p.db.Query(query, updateValues...).Exec(); err != nil {
		return nil, err
	}
	return group, nil
}

// DeleteScimGroup removes a SCIM group record.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.ScimGroup)
	return p.db.Query(query, group.ID).Exec()
}

// GetScimGroupByID fetches a SCIM group by primary key.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", scimGroupColumns, KeySpace+"."+schemas.Collections.ScimGroup)
	if err := scanScimGroup(p.db.Query(query, id).Consistency(gocql.One).Scan, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org. displayName is compared case-insensitively:
// SCIM Group.displayName is caseExact:false (RFC 7644 §3.4.2.2). CQL cannot
// apply a function like LOWER() to a column in a WHERE clause, so scope the
// query to the org and compare displayName with strings.EqualFold in Go (an
// org's group set is small), mirroring the DynamoDB fetch-then-filter shape.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	// No LIMIT here deliberately: a CQL LIMIT truncates the combined result set
	// across every page, which would silently drop a match beyond it — the
	// exact bug groupScanSafetyCap exists to avoid. A real match anywhere in the
	// org's group set is still found; the safety cap below bounds the Go-side
	// loop instead of the CQL query.
	//
	// PageSize is set explicitly (rather than leaving gocql's session default)
	// because with none set, ALLOW FILTERING fetches the ENTIRE result in one
	// frame — the Scanner would just iterate an already-fully-materialized
	// buffer, not genuinely page. Setting it here makes the driver fetch and
	// examine the partition incrementally, so an early match (the common case)
	// avoids pulling the rest of a large org's rows into memory at all, and the
	// safety cap actually bounds real fetch cost, not just a client-side count
	// over data that was already all in memory.
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? ALLOW FILTERING", scimGroupColumns, KeySpace+"."+schemas.Collections.ScimGroup)
	scanner := p.db.Query(query, orgID).Consistency(gocql.One).PageSize(cassandraGroupPageSize).Iter().Scanner()
	examined := 0
	for scanner.Next() {
		var group schemas.ScimGroup
		if err := scanScimGroup(scanner.Scan, &group); err != nil {
			return nil, err
		}
		if strings.EqualFold(group.DisplayName, displayName) {
			return &group, nil
		}
		examined++
		if examined >= groupScanSafetyCap {
			p.dependencies.Log.Warn().Str("org_id", orgID).Int("examined", examined).
				Msg("GetScimGroupByOrgAndDisplayName: hit the scan safety cap without a match")
			return nil, gocql.ErrNotFound
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, gocql.ErrNotFound
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId within an org. externalId is stored org-namespaced ("<orgID>:<raw>")
// exactly like User.ExternalID, so this can never resolve another org's group.
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? AND external_id = ? LIMIT 1 ALLOW FILTERING", scimGroupColumns, KeySpace+"."+schemas.Collections.ScimGroup)
	if err := scanScimGroup(p.db.Query(query, orgID, orgID+":"+externalID).Consistency(gocql.One).Scan, &group); err != nil {
		return nil, err
	}
	return &group, nil
}
