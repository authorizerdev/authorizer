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

// --- SAMLServiceProvider (registered downstream SPs; Authorizer as IdP) ---

const samlServiceProviderColumns = "id, org_id, name, entity_id, acs_url, sp_cert_pem, name_id_format, mapped_attributes, allow_idp_initiated, is_active, created_at, updated_at"

// scanSAMLServiceProvider maps the samlServiceProviderColumns projection onto a struct.
func scanSAMLServiceProvider(scan func(...interface{}) error, sp *schemas.SAMLServiceProvider) error {
	return scan(&sp.ID, &sp.OrgID, &sp.Name, &sp.EntityID, &sp.ACSURL, &sp.SPCertPEM, &sp.NameIDFormat, &sp.MappedAttributes, &sp.AllowIDPInitiated, &sp.IsActive, &sp.CreatedAt, &sp.UpdatedAt)
}

// AddSAMLServiceProvider registers a new downstream SP.
func (p *provider) AddSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.ID == "" {
		sp.ID = uuid.New().String()
	}
	sp.Key = sp.ID
	now := time.Now().Unix()
	sp.CreatedAt = now
	sp.UpdatedAt = now
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.SAMLServiceProvider, samlServiceProviderColumns)
	err := p.db.Query(insertQuery, sp.ID, sp.OrgID, sp.Name, sp.EntityID, sp.ACSURL, sp.SPCertPEM, sp.NameIDFormat, sp.MappedAttributes, sp.AllowIDPInitiated, sp.IsActive, sp.CreatedAt, sp.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// UpdateSAMLServiceProvider updates a registered SP.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	spMap := buildCQLColumnMap(sp)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range spMap {
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
	updateValues = append(updateValues, sp.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.SAMLServiceProvider, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.SAMLServiceProvider)
	return p.db.Query(query, sp.ID).Exec()
}

// GetSAMLServiceProviderByID fetches a registered SP by primary key.
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	var sp schemas.SAMLServiceProvider
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", samlServiceProviderColumns, KeySpace+"."+schemas.Collections.SAMLServiceProvider)
	err := scanSAMLServiceProvider(p.db.Query(query, id).Consistency(gocql.One).Scan, &sp)
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// GetSAMLServiceProviderByOrgAndEntityID resolves the single registered SP for an
// (orgID, entityID) pair — the AuthnRequest-Issuer → trusted-ACS binding.
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	var sp schemas.SAMLServiceProvider
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? AND entity_id = ? LIMIT 1 ALLOW FILTERING", samlServiceProviderColumns, KeySpace+"."+schemas.Collections.SAMLServiceProvider)
	err := scanSAMLServiceProvider(p.db.Query(query, orgID, entityID).Consistency(gocql.One).Scan, &sp)
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// ListSAMLServiceProviders returns the registered SPs for an org (paginated).
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	sps := []*schemas.SAMLServiceProvider{}
	paginationClone := pagination
	table := KeySpace + "." + schemas.Collections.SAMLServiceProvider

	totalCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE org_id = ? ALLOW FILTERING", table)
	err := p.db.Query(totalCountQuery, orgID).Consistency(gocql.One).Scan(&paginationClone.Total)
	if err != nil {
		return nil, nil, err
	}

	// there is no offset in cassandra
	// so we fetch till limit + offset
	// and return the results from offset to limit
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? LIMIT %d ALLOW FILTERING", samlServiceProviderColumns, table, pagination.Limit+pagination.Offset)
	scanner := p.db.Query(query, orgID).Iter().Scanner()
	counter := int64(0)
	for scanner.Next() {
		if counter >= pagination.Offset {
			var sp schemas.SAMLServiceProvider
			if err := scanSAMLServiceProvider(scanner.Scan, &sp); err != nil {
				return nil, nil, err
			}
			sps = append(sps, &sp)
		}
		counter++
	}
	return sps, paginationClone, nil
}

// --- SAMLIDPKey (per-org signing keypairs with rotation) ---

const samlIDPKeyColumns = "id, org_id, cert_pem, private_key_enc, algorithm, status, created_at, updated_at"

// scanSAMLIDPKey maps the samlIDPKeyColumns projection onto a struct.
func scanSAMLIDPKey(scan func(...interface{}) error, key *schemas.SAMLIDPKey) error {
	return scan(&key.ID, &key.OrgID, &key.CertPEM, &key.PrivateKeyEnc, &key.Algorithm, &key.Status, &key.CreatedAt, &key.UpdatedAt)
}

// AddSAMLIDPKey persists a newly-generated signing keypair.
func (p *provider) AddSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.ID == "" {
		key.ID = uuid.New().String()
	}
	key.Key = key.ID
	now := time.Now().Unix()
	key.CreatedAt = now
	key.UpdatedAt = now
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.SAMLIDPKey, samlIDPKeyColumns)
	err := p.db.Query(insertQuery, key.ID, key.OrgID, key.CertPEM, key.PrivateKeyEnc, key.Algorithm, key.Status, key.CreatedAt, key.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return key, nil
}

// UpdateSAMLIDPKey updates a signing key (used to flip status).
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	keyMap := buildCQLColumnMap(key)
	updateFields := ""
	var updateValues []interface{}
	for k, value := range keyMap {
		if k == "id" || k == "_key" {
			continue
		}
		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", k)
			continue
		}
		updateFields += fmt.Sprintf("%s = ?, ", k)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	updateValues = append(updateValues, key.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.SAMLIDPKey, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteSAMLIDPKey removes a signing key.
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.SAMLIDPKey)
	return p.db.Query(query, key.ID).Exec()
}

// GetSAMLIDPKeyByID fetches a signing key by primary key.
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	var key schemas.SAMLIDPKey
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", samlIDPKeyColumns, KeySpace+"."+schemas.Collections.SAMLIDPKey)
	err := scanSAMLIDPKey(p.db.Query(query, id).Consistency(gocql.One).Scan, &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// ListSAMLIDPKeys returns every signing key for an org.
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	keys := []*schemas.SAMLIDPKey{}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? ALLOW FILTERING", samlIDPKeyColumns, KeySpace+"."+schemas.Collections.SAMLIDPKey)
	// ponytail: Cassandra can't ORDER BY a non-clustering column, so the SQL
	// side's created_at DESC ordering isn't replicated here; callers that need
	// ordering sort in-memory (key sets are tiny — a handful per org).
	scanner := p.db.Query(query, orgID).Iter().Scanner()
	for scanner.Next() {
		var key schemas.SAMLIDPKey
		if err := scanSAMLIDPKey(scanner.Scan, &key); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}
