package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const serviceAccountColumns = "_id, name, description, client_secret, allowed_scopes, is_active, created_at, updated_at"

// AddServiceAccount creates a new service account record.
func (p *provider) AddServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.ServiceAccount).Insert(sa.ID, sa, &insertOpt)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateServiceAccount: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	bytes, err := json.Marshal(sa)
	if err != nil {
		return nil, err
	}
	// use decoder instead of json.Unmarshall, because it converts int64 -> float64 after unmarshalling
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	saMap := map[string]interface{}{}
	err = decoder.Decode(&saMap)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(saMap)
	params["_id"] = sa.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.ServiceAccount, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteServiceAccount removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.ServiceAccount).Remove(sa.ID, &removeOpt)
	if err != nil {
		return err
	}
	params := make(map[string]interface{}, 1)
	params["service_account_id"] = sa.ID
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE service_account_id=$service_account_id`, p.scopeName, schemas.Collections.TrustedIssuer)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return err
	}
	return nil
}

// GetServiceAccountByID fetches a service account by primary key.
func (p *provider) GetServiceAccountByID(ctx context.Context, id string) (*schemas.ServiceAccount, error) {
	var sa *schemas.ServiceAccount
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, serviceAccountColumns, p.scopeName, schemas.Collections.ServiceAccount)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&sa)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// ListServiceAccounts returns a paginated list of service accounts.
func (p *provider) ListServiceAccounts(ctx context.Context, pagination *model.Pagination) ([]*schemas.ServiceAccount, *model.Pagination, error) {
	serviceAccounts := []*schemas.ServiceAccount{}
	paginationClone := pagination
	params := make(map[string]interface{}, 2)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.ServiceAccount)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", serviceAccountColumns, p.scopeName, schemas.Collections.ServiceAccount)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, nil, err
	}
	for queryResult.Next() {
		var sa schemas.ServiceAccount
		err := queryResult.Row(&sa)
		if err != nil {
			log.Fatal(err)
		}
		serviceAccounts = append(serviceAccounts, &sa)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return serviceAccounts, paginationClone, nil
}
