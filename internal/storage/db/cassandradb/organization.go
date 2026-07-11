package cassandradb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const organizationColumns = "id, name, display_name, enabled, created_at, updated_at"

// scanOrganization maps the organizationColumns projection onto a struct.
func scanOrganization(scan func(...interface{}) error, org *schemas.Organization) error {
	return scan(&org.ID, &org.Name, &org.DisplayName, &org.Enabled, &org.CreatedAt, &org.UpdatedAt)
}

// AddOrganization creates a new organization record.
func (p *provider) AddOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	org.Key = org.ID
	now := time.Now().Unix()
	org.CreatedAt = now
	org.UpdatedAt = now
	// name is unique (gorm uniqueIndex on the SQL side). Cassandra has no
	// cross-attribute unique constraint, so guard with a check-then-insert
	// mirroring AddTrustedIssuer's issuer_url pre-check.
	// ponytail: inherent TOCTOU race — two concurrent inserts of the same name
	// can both pass. Closes the sequential admin-misconfiguration case only;
	// a race-free guard would need an LWT on a name-keyed table.
	if existing, _ := p.GetOrganizationByName(ctx, org.Name); existing != nil {
		return nil, fmt.Errorf("organization with %s name already exists", org.Name)
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.Organization, organizationColumns)
	err := p.db.Query(insertQuery, org.ID, org.Name, org.DisplayName, org.Enabled, org.CreatedAt, org.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return org, nil
}

// UpdateOrganization updates an organization record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateOrganization(ctx context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	if org.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrganization: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	org.UpdatedAt = time.Now().Unix()
	orgMap := buildCQLColumnMap(org)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range orgMap {
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
	updateValues = append(updateValues, org.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Organization, updateFields)
	if err := p.db.Query(query, updateValues...).Exec(); err != nil {
		return nil, err
	}
	return org, nil
}

// DeleteOrganization removes an organization and all its memberships.
// Mirrors the DeleteClient cascade-delete pattern.
func (p *provider) DeleteOrganization(ctx context.Context, org *schemas.Organization) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Organization)
	if err := p.db.Query(query, org.ID).Exec(); err != nil {
		return err
	}

	getMembershipsQuery := fmt.Sprintf("SELECT id FROM %s WHERE org_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.OrgMembership)
	scanner := p.db.Query(getMembershipsQuery, org.ID).Iter().Scanner()
	var membershipIDList []string
	for scanner.Next() {
		var membershipID string
		if err := scanner.Scan(&membershipID); err != nil {
			return err
		}
		membershipIDList = append(membershipIDList, membershipID)
	}
	if len(membershipIDList) > 0 {
		placeholders := strings.Repeat("?,", len(membershipIDList))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(membershipIDList))
		for i, id := range membershipIDList {
			deleteValues[i] = id
		}
		query = fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.OrgMembership, placeholders)
		if err := p.db.Query(query, deleteValues...).Exec(); err != nil {
			return err
		}
	}
	// Cascade verified domains — otherwise the domain becomes permanently
	// unclaimable (it is the unique partition key of org_domains).
	return p.DeleteOrgDomainsByOrg(ctx, org.ID)
}

// GetOrganizationByID fetches an organization by primary key.
func (p *provider) GetOrganizationByID(ctx context.Context, id string) (*schemas.Organization, error) {
	var org schemas.Organization
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", organizationColumns, KeySpace+"."+schemas.Collections.Organization)
	if err := scanOrganization(p.db.Query(query, id).Consistency(gocql.One).Scan, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// GetOrganizationByName fetches an organization by its unique name slug.
func (p *provider) GetOrganizationByName(ctx context.Context, name string) (*schemas.Organization, error) {
	var org schemas.Organization
	query := fmt.Sprintf("SELECT %s FROM %s WHERE name = ? LIMIT 1 ALLOW FILTERING", organizationColumns, KeySpace+"."+schemas.Collections.Organization)
	if err := scanOrganization(p.db.Query(query, name).Consistency(gocql.One).Scan, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// ListOrganizations returns a paginated list of organizations.
func (p *provider) ListOrganizations(ctx context.Context, pagination *model.Pagination) ([]*schemas.Organization, *model.Pagination, error) {
	orgs := []*schemas.Organization{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.Organization)
	if err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total); err != nil {
		return nil, nil, err
	}
	// there is no offset in cassandra: fetch limit + offset, return offset..limit
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT %d", organizationColumns, KeySpace+"."+schemas.Collections.Organization, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var org schemas.Organization
			if err := scanOrganization(scanner.Scan, &org); err != nil {
				return nil, nil, err
			}
			orgs = append(orgs, &org)
		}
		counter++
	}
	return orgs, paginationClone, nil
}
