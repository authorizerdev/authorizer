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

const orgMembershipColumns = "id, org_id, user_id, roles, created_at, updated_at"

// orgMembershipPK derives the membership's primary key deterministically from
// (org_id, user_id). ScyllaDB secondary indexes are materialized-view backed
// and update ASYNCHRONOUSLY, so a lookup by the org_id/user_id index right after
// an insert can miss the just-written row (the multi-member add-then-get race).
// Keying the base table on a deterministic id lets GetOrgMembership read by
// primary key — synchronous, index-free, and race-free — while (org_id,user_id)
// uniqueness falls out for free (the same pair always maps to the same key).
func orgMembershipPK(orgID, userID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(orgID+":"+userID)).String()
}

// scanOrgMembership maps the orgMembershipColumns projection onto a struct.
func scanOrgMembership(scan func(...interface{}) error, membership *schemas.OrgMembership) error {
	return scan(&membership.ID, &membership.OrgID, &membership.UserID, &membership.Roles, &membership.CreatedAt, &membership.UpdatedAt)
}

// AddOrgMembership creates a new membership. (org_id, user_id) is unique.
// Cassandra has no cross-attribute unique constraint, so guard with a
// check-then-insert mirroring AddTrustedIssuer's pre-check.
// ponytail: inherent TOCTOU race — closes the sequential case only.
func (p *provider) AddOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	// Deterministic PK from (org_id, user_id) so the row is readable by primary
	// key immediately after insert — see orgMembershipPK. Overrides any incoming
	// id; the id is opaque and no caller depends on a specific value.
	membership.ID = orgMembershipPK(membership.OrgID, membership.UserID)
	membership.Key = membership.ID
	now := time.Now().Unix()
	membership.CreatedAt = now
	membership.UpdatedAt = now
	if existing, _ := p.GetOrgMembership(ctx, membership.OrgID, membership.UserID); existing != nil {
		return nil, fmt.Errorf("membership for org_id %s and user_id %s already exists", membership.OrgID, membership.UserID)
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.OrgMembership, orgMembershipColumns)
	err := p.db.Query(insertQuery, membership.ID, membership.OrgID, membership.UserID, membership.Roles, membership.CreatedAt, membership.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// UpdateOrgMembership updates a membership record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateOrgMembership(ctx context.Context, membership *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if membership.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateOrgMembership: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	membership.UpdatedAt = time.Now().Unix()
	membershipMap := buildCQLColumnMap(membership)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range membershipMap {
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
	updateValues = append(updateValues, membership.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.OrgMembership, updateFields)
	if err := p.db.Query(query, updateValues...).Exec(); err != nil {
		return nil, err
	}
	return membership, nil
}

// DeleteOrgMembership removes a membership record.
func (p *provider) DeleteOrgMembership(ctx context.Context, membership *schemas.OrgMembership) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.OrgMembership)
	return p.db.Query(query, membership.ID).Exec()
}

// GetOrgMembership fetches the membership for a (orgID, userID) pair.
func (p *provider) GetOrgMembership(ctx context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	var membership schemas.OrgMembership
	// Read by the deterministic primary key (base table) — NOT the async org_id/
	// user_id secondary index — so a membership is visible immediately after it
	// is written (fixes the ScyllaDB add-then-get race).
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", orgMembershipColumns, KeySpace+"."+schemas.Collections.OrgMembership)
	if err := scanOrgMembership(p.db.Query(query, orgMembershipPK(orgID, userID)).Consistency(gocql.One).Scan, &membership); err != nil {
		return nil, err
	}
	return &membership, nil
}

// ListOrgMembershipsByOrg returns paginated memberships of an organization.
func (p *provider) ListOrgMembershipsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "org_id", orgID, pagination)
}

// ListOrgMembershipsByUser returns paginated memberships held by a user.
func (p *provider) ListOrgMembershipsByUser(ctx context.Context, userID string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return p.listOrgMemberships(ctx, "user_id", userID, pagination)
}

func (p *provider) listOrgMemberships(ctx context.Context, filterColumn, filterValue string, pagination *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	memberships := []*schemas.OrgMembership{}
	paginationClone := pagination
	table := KeySpace + "." + schemas.Collections.OrgMembership

	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ? ALLOW FILTERING", table, filterColumn)
	if err := p.db.Query(totalCountQuery, filterValue).Consistency(gocql.One).Scan(&paginationClone.Total); err != nil {
		return nil, nil, err
	}

	// there is no offset in cassandra: fetch limit + offset, return offset..limit
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? LIMIT %d ALLOW FILTERING", orgMembershipColumns, table, filterColumn, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query, filterValue).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var membership schemas.OrgMembership
			if err := scanOrgMembership(scanner.Scan, &membership); err != nil {
				return nil, nil, err
			}
			memberships = append(memberships, &membership)
		}
		counter++
	}
	return memberships, paginationClone, nil
}
