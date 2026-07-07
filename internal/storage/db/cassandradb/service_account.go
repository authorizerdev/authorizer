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

const serviceAccountColumns = "id, name, description, client_secret, allowed_scopes, is_active, created_at, updated_at"

// AddServiceAccount creates a new service account record.
func (p *provider) AddServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.ServiceAccount, serviceAccountColumns)
	err := p.db.Query(insertQuery, sa.ID, sa.Name, sa.Description, sa.ClientSecret, sa.AllowedScopes, sa.IsActive, sa.CreatedAt, sa.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateServiceAccount: caller must load record before updating (CreatedAt is zero — partial struct detected)")
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
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.ServiceAccount, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteServiceAccount removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.ServiceAccount)
	err := p.db.Query(query, sa.ID).Exec()
	if err != nil {
		return err
	}

	getIssuersQuery := fmt.Sprintf("SELECT id FROM %s WHERE service_account_id = ? ALLOW FILTERING", KeySpace+"."+schemas.Collections.TrustedIssuer)
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

// GetServiceAccountByID fetches a service account by primary key.
func (p *provider) GetServiceAccountByID(ctx context.Context, id string) (*schemas.ServiceAccount, error) {
	var sa schemas.ServiceAccount
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", serviceAccountColumns, KeySpace+"."+schemas.Collections.ServiceAccount)
	err := p.db.Query(query, id).Consistency(gocql.One).Scan(&sa.ID, &sa.Name, &sa.Description, &sa.ClientSecret, &sa.AllowedScopes, &sa.IsActive, &sa.CreatedAt, &sa.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// ListServiceAccounts returns a paginated list of service accounts.
func (p *provider) ListServiceAccounts(ctx context.Context, pagination *model.Pagination) ([]*schemas.ServiceAccount, *model.Pagination, error) {
	serviceAccounts := []*schemas.ServiceAccount{}
	paginationClone := pagination
	totalCountQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, KeySpace+"."+schemas.Collections.ServiceAccount)
	err := p.db.Query(totalCountQuery).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}
	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT %d", serviceAccountColumns, KeySpace+"."+schemas.Collections.ServiceAccount, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var sa schemas.ServiceAccount
			err := scanner.Scan(&sa.ID, &sa.Name, &sa.Description, &sa.ClientSecret, &sa.AllowedScopes, &sa.IsActive, &sa.CreatedAt, &sa.UpdatedAt)
			if err != nil {
				return nil, nil, err
			}
			serviceAccounts = append(serviceAccounts, &sa)
		}
		counter++
	}
	return serviceAccounts, paginationClone, nil
}
