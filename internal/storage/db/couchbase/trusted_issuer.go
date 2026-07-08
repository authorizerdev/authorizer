package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const trustedIssuerColumns = "_id, client_id, kind, org_id, name, issuer_url, key_source_type, jwks_url, expected_aud, subject_claim, allowed_subjects, issuer_type, auth_method, is_active, enable_token_review, kubernetes_api_server_url, spiffe_refresh_hint_seconds, trusted_proxy_header, trusted_proxy_cidrs, sso_client_id, sso_client_secret_enc, sso_scopes, sso_redirect_uri, created_at, updated_at"

// AddTrustedIssuer creates a new trusted issuer record.
func (p *provider) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.ID == "" {
		issuer.ID = uuid.New().String()
	}
	issuer.Key = issuer.ID
	now := time.Now().Unix()
	issuer.CreatedAt = now
	issuer.UpdatedAt = now
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(issuer)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.TrustedIssuer).Insert(issuer.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// UpdateTrustedIssuer updates a trusted issuer record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	issuerMap, err := structToDocument(issuer)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(issuerMap)
	params["_id"] = issuer.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.TrustedIssuer, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer record.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.TrustedIssuer).Remove(issuer.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, trustedIssuerColumns, p.scopeName, schemas.Collections.TrustedIssuer)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	issuer := &schemas.TrustedIssuer{}
	if err := decodeDocument(raw, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// Called on every client_assertion validation — served by the issuer_url index.
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	params := make(map[string]interface{}, 1)
	params["issuer_url"] = issuerURL
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE issuer_url=$issuer_url LIMIT 1`, trustedIssuerColumns, p.scopeName, schemas.Collections.TrustedIssuer)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	issuer := &schemas.TrustedIssuer{}
	if err := decodeDocument(raw, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// GetTrustedIssuerByOrgIDAndKind fetches an organization's SSO connection by its
// (org_id, kind) pair — the lookup that resolves an org's sso_oidc/sso_saml row.
func (p *provider) GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID, kind string) (*schemas.TrustedIssuer, error) {
	params := make(map[string]interface{}, 2)
	params["org_id"] = orgID
	params["kind"] = kind
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id AND kind=$kind LIMIT 1`, trustedIssuerColumns, p.scopeName, schemas.Collections.TrustedIssuer)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	issuer := &schemas.TrustedIssuer{}
	if err := decodeDocument(raw, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	issuers := []*schemas.TrustedIssuer{}
	paginationClone := pagination
	table := fmt.Sprintf("%s.%s", p.scopeName, schemas.Collections.TrustedIssuer)

	whereClause := ""
	params := make(map[string]interface{}, 3)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	if serviceAccountID != "" {
		whereClause = " WHERE client_id=$client_id"
		params["client_id"] = serviceAccountID
	}

	countParams := make(map[string]interface{}, 1)
	if serviceAccountID != "" {
		countParams["client_id"] = serviceAccountID
	}
	total := TotalDocs{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as Total FROM %s%s", table, whereClause)
	countRes, err := p.db.Query(countQuery, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: countParams,
	})
	if err != nil {
		return nil, nil, err
	}
	_ = countRes.One(&total)
	paginationClone.Total = total.Total

	query := fmt.Sprintf("SELECT %s FROM %s%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", trustedIssuerColumns, table, whereClause)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, nil, err
		}
		issuer := &schemas.TrustedIssuer{}
		if err := decodeDocument(raw, issuer); err != nil {
			return nil, nil, err
		}
		issuers = append(issuers, issuer)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return issuers, paginationClone, nil
}
