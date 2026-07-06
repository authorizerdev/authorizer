package cassandradb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const trustedIssuerColumns = "id, service_account_id, name, issuer_url, key_source_type, jwks_url, expected_aud, subject_claim, issuer_type, auth_method, is_active, enable_token_review, kubernetes_api_server_url, spiffe_refresh_hint_seconds, trusted_proxy_header, trusted_proxy_cidrs, created_at, updated_at"

// scanTrustedIssuer maps the trustedIssuerColumns projection onto a struct.
func scanTrustedIssuer(scan func(...interface{}) error, issuer *schemas.TrustedIssuer) error {
	return scan(&issuer.ID, &issuer.ServiceAccountID, &issuer.Name, &issuer.IssuerURL, &issuer.KeySourceType, &issuer.JWKSUrl, &issuer.ExpectedAud, &issuer.SubjectClaim, &issuer.IssuerType, &issuer.AuthMethod, &issuer.IsActive, &issuer.EnableTokenReview, &issuer.KubernetesAPIServerURL, &issuer.SpiffeRefreshHintSeconds, &issuer.TrustedProxyHeader, &issuer.TrustedProxyCIDRs, &issuer.CreatedAt, &issuer.UpdatedAt)
}

// AddTrustedIssuer creates a new trusted issuer record.
func (p *provider) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.ID == "" {
		issuer.ID = uuid.New().String()
	}
	issuer.Key = issuer.ID
	now := time.Now().Unix()
	issuer.CreatedAt = now
	issuer.UpdatedAt = now
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.TrustedIssuer, trustedIssuerColumns)
	err := p.db.Query(insertQuery, issuer.ID, issuer.ServiceAccountID, issuer.Name, issuer.IssuerURL, issuer.KeySourceType, issuer.JWKSUrl, issuer.ExpectedAud, issuer.SubjectClaim, issuer.IssuerType, issuer.AuthMethod, issuer.IsActive, issuer.EnableTokenReview, issuer.KubernetesAPIServerURL, issuer.SpiffeRefreshHintSeconds, issuer.TrustedProxyHeader, issuer.TrustedProxyCIDRs, issuer.CreatedAt, issuer.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// UpdateTrustedIssuer updates a trusted issuer record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(issuer)
	if err != nil {
		return nil, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	issuerMap := map[string]interface{}{}
	err = decoder.Decode(&issuerMap)
	if err != nil {
		return nil, err
	}
	convertMapValues(issuerMap)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range issuerMap {
		if key == "_id" {
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
	updateValues = append(updateValues, issuer.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.TrustedIssuer, updateFields)
	err = p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer record.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.TrustedIssuer)
	return p.db.Query(query, issuer.ID).Exec()
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	var issuer schemas.TrustedIssuer
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", trustedIssuerColumns, KeySpace+"."+schemas.Collections.TrustedIssuer)
	err := scanTrustedIssuer(p.db.Query(query, id).Consistency(gocql.One).Scan, &issuer)
	if err != nil {
		return nil, err
	}
	return &issuer, nil
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// Called on every client_assertion validation — served by the issuer_url secondary index.
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	var issuer schemas.TrustedIssuer
	query := fmt.Sprintf("SELECT %s FROM %s WHERE issuer_url = ? LIMIT 1 ALLOW FILTERING", trustedIssuerColumns, KeySpace+"."+schemas.Collections.TrustedIssuer)
	err := scanTrustedIssuer(p.db.Query(query, issuerURL).Consistency(gocql.One).Scan, &issuer)
	if err != nil {
		return nil, err
	}
	return &issuer, nil
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	issuers := []*schemas.TrustedIssuer{}
	paginationClone := pagination
	table := KeySpace + "." + schemas.Collections.TrustedIssuer

	whereClause := ""
	var filterValues []interface{}
	if serviceAccountID != "" {
		whereClause = " WHERE service_account_id = ? ALLOW FILTERING"
		filterValues = append(filterValues, serviceAccountID)
	}

	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", table, whereClause)
	err := p.db.Query(totalCountQuery, filterValues...).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	var query string
	if serviceAccountID != "" {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE service_account_id = ? LIMIT %d ALLOW FILTERING", trustedIssuerColumns, table, pagination.Limit+pagination.Offset)
	} else {
		query = fmt.Sprintf("SELECT %s FROM %s LIMIT %d", trustedIssuerColumns, table, pagination.Limit+pagination.Offset)
	}
	scanner := p.db.Query(query, filterValues...).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var issuer schemas.TrustedIssuer
			if err := scanTrustedIssuer(scanner.Scan, &issuer); err != nil {
				return nil, nil, err
			}
			issuers = append(issuers, &issuer)
		}
		counter++
	}
	return issuers, paginationClone, nil
}
