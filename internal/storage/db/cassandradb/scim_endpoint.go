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

const scimEndpointColumns = "id, org_id, token_hash, enabled, created_at, updated_at"

// scanScimEndpoint maps the scimEndpointColumns projection onto a struct.
func scanScimEndpoint(scan func(...interface{}) error, endpoint *schemas.ScimEndpoint) error {
	return scan(&endpoint.ID, &endpoint.OrgID, &endpoint.TokenHash, &endpoint.Enabled, &endpoint.CreatedAt, &endpoint.UpdatedAt)
}

// AddScimEndpoint creates a new SCIM endpoint. OrgID is unique — one endpoint
// per org. Cassandra has no cross-attribute unique constraint, so guard with a
// check-then-insert mirroring AddOrganization's name pre-check.
// ponytail: inherent TOCTOU race — closes the sequential case only.
func (p *provider) AddScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	endpoint.Key = endpoint.ID
	now := time.Now().Unix()
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now
	if existing, _ := p.GetScimEndpointByOrgID(ctx, endpoint.OrgID); existing != nil {
		return nil, fmt.Errorf("scim endpoint for org_id %s already exists", endpoint.OrgID)
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.ScimEndpoint, scimEndpointColumns)
	err := p.db.Query(insertQuery, endpoint.ID, endpoint.OrgID, endpoint.TokenHash, endpoint.Enabled, endpoint.CreatedAt, endpoint.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return endpoint, nil
}

// UpdateScimEndpoint updates a SCIM endpoint record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if endpoint.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	endpoint.UpdatedAt = time.Now().Unix()
	endpointMap := buildCQLColumnMap(endpoint)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range endpointMap {
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
	updateValues = append(updateValues, endpoint.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.ScimEndpoint, updateFields)
	if err := p.db.Query(query, updateValues...).Exec(); err != nil {
		return nil, err
	}
	return endpoint, nil
}

// DeleteScimEndpoint removes a SCIM endpoint record.
func (p *provider) DeleteScimEndpoint(ctx context.Context, endpoint *schemas.ScimEndpoint) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.ScimEndpoint)
	return p.db.Query(query, endpoint.ID).Exec()
}

// GetScimEndpointByID fetches a SCIM endpoint by primary key.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	var endpoint schemas.ScimEndpoint
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", scimEndpointColumns, KeySpace+"."+schemas.Collections.ScimEndpoint)
	if err := scanScimEndpoint(p.db.Query(query, id).Consistency(gocql.One).Scan, &endpoint); err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// GetScimEndpointByOrgID fetches a SCIM endpoint by its unique org ID.
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	var endpoint schemas.ScimEndpoint
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? LIMIT 1 ALLOW FILTERING", scimEndpointColumns, KeySpace+"."+schemas.Collections.ScimEndpoint)
	if err := scanScimEndpoint(p.db.Query(query, orgID).Consistency(gocql.One).Scan, &endpoint); err != nil {
		return nil, err
	}
	return &endpoint, nil
}
