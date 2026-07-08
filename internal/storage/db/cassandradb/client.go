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

const clientColumns = "id, client_id, kind, name, description, client_secret, allowed_scopes, redirect_uris, grant_types, token_endpoint_auth_method, is_active, org_id, created_at, updated_at"

// AddClient creates a new service account record.
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	if sa.ClientID == "" {
		sa.ClientID = sa.ID
	}
	// Cassandra has no cross-attribute unique constraint, so guard client_id
	// uniqueness with a check-then-insert mirroring AddTrustedIssuer's issuer_url
	// pre-check.
	// ponytail: inherent TOCTOU race — two concurrent inserts of the same
	// client_id can both pass this check. Cassandra offers no atomic IF NOT
	// EXISTS on a non-partition-key column; this closes the common case
	// (sequential admin/boot-seed) only. A fully race-free guard would need an
	// LWT on a dedicated client_id-keyed table.
	if existing, _ := p.GetClientByClientID(ctx, sa.ClientID); existing != nil {
		return nil, fmt.Errorf("client with client_id %s already exists", sa.ClientID)
	}
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.Client, clientColumns)
	err := p.db.Query(insertQuery, sa.ID, sa.ClientID, sa.Kind, sa.Name, sa.Description, sa.ClientSecret, sa.AllowedScopes, sa.RedirectURIs, sa.GrantTypes, sa.TokenEndpointAuthMethod, sa.IsActive, sa.OrgID, sa.CreatedAt, sa.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateClient updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	// Column names are sourced from the `cql` struct tag (not json.Marshal, which
	// drops json:"-" fields such as client_secret — see buildCQLColumnMap). Without
	// this, secret rotation silently no-op'd: client_secret was never in the SET
	// clause, so the old hash stayed active forever.
	saMap := buildCQLColumnMap(sa)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range saMap {
		if key == "id" {
			continue
		}
		if key == "_key" {
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
	updateValues = append(updateValues, sa.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.Client, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteClient removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.Client)
	err := p.db.Query(query, sa.ID).Exec()
	if err != nil {
		return err
	}

	getIssuersQuery := fmt.Sprintf("SELECT id FROM %s WHERE client_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.TrustedIssuer)
	scanner := p.db.Query(getIssuersQuery, sa.ID).Iter().Scanner()
	var issuerIDList []string
	for scanner.Next() {
		var issuerID string
		if err := scanner.Scan(&issuerID); err != nil {
			return err
		}
		issuerIDList = append(issuerIDList, issuerID)
	}
	if len(issuerIDList) > 0 {
		placeholders := strings.Repeat("?,", len(issuerIDList))
		placeholders = strings.TrimSuffix(placeholders, ",")
		deleteValues := make([]interface{}, len(issuerIDList))
		for i, id := range issuerIDList {
			deleteValues[i] = id
		}
		query = fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", KeySpace+"."+schemas.Collections.TrustedIssuer, placeholders)
		if err := p.db.Query(query, deleteValues...).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetClientByID fetches a service account by primary key.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	var sa schemas.Client
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", clientColumns, KeySpace+"."+schemas.Collections.Client)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(&sa.ID, &sa.ClientID, &sa.Kind, &sa.Name, &sa.Description, &sa.ClientSecret, &sa.AllowedScopes, &sa.RedirectURIs, &sa.GrantTypes, &sa.TokenEndpointAuthMethod, &sa.IsActive, &sa.OrgID, &sa.CreatedAt, &sa.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// GetClientByClientID fetches a client by its unique public client_id.
// Served by the client_id secondary index.
func (p *provider) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	var sa schemas.Client
	query := fmt.Sprintf("SELECT %s FROM %s WHERE client_id = ? LIMIT 1 ALLOW FILTERING", clientColumns, KeySpace+"."+schemas.Collections.Client)
	err := p.db.Query(query, clientID).Consistency(gocql.One).Scan(&sa.ID, &sa.ClientID, &sa.Kind, &sa.Name, &sa.Description, &sa.ClientSecret, &sa.AllowedScopes, &sa.RedirectURIs, &sa.GrantTypes, &sa.TokenEndpointAuthMethod, &sa.IsActive, &sa.OrgID, &sa.CreatedAt, &sa.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// ListClients returns a paginated list of service accounts.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	clients := []*schemas.Client{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.Client)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT %d", clientColumns, KeySpace+"."+schemas.Collections.Client, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var sa schemas.Client
			err := scanner.Scan(&sa.ID, &sa.ClientID, &sa.Kind, &sa.Name, &sa.Description, &sa.ClientSecret, &sa.AllowedScopes, &sa.RedirectURIs, &sa.GrantTypes, &sa.TokenEndpointAuthMethod, &sa.IsActive, &sa.OrgID, &sa.CreatedAt, &sa.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			clients = append(clients, &sa)
		}
		counter++
	}
	return clients, paginationClone, nil
}
