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
// displayName within an org.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	var group schemas.ScimGroup
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? AND display_name = ? LIMIT 1 ALLOW FILTERING", scimGroupColumns, KeySpace+"."+schemas.Collections.ScimGroup)
	if err := scanScimGroup(p.db.Query(query, orgID, displayName).Consistency(gocql.One).Scan, &group); err != nil {
		return nil, err
	}
	return &group, nil
}
