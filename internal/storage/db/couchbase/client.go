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

const clientColumns = "_id, kind, name, description, client_secret, allowed_scopes, is_active, created_at, updated_at"

// AddClient creates a new service account record.
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
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
	doc, err := structToDocument(sa)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.Client).Insert(sa.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateClient updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	saMap, err := structToDocument(sa)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(saMap)
	params["_id"] = sa.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.Client, updateFields)
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

// DeleteClient removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Client).Remove(sa.ID, &removeOpt)
	if err != nil {
		return err
	}
	params := make(map[string]interface{}, 1)
	params["client_id"] = sa.ID
	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE client_id=$client_id`, p.scopeName, schemas.Collections.TrustedIssuer)
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

// GetClientByID fetches a service account by primary key.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, clientColumns, p.scopeName, schemas.Collections.Client)
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
	sa := &schemas.Client{}
	if err := decodeDocument(raw, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// ListClients returns a paginated list of service accounts.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	clients := []*schemas.Client{}
	paginationClone := pagination
	params := make(map[string]interface{}, 2)
	params["offset"] = paginationClone.Offset
	params["limit"] = paginationClone.Limit
	total, err := p.GetTotalDocs(ctx, schemas.Collections.Client)
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = total
	query := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY created_at DESC OFFSET $offset LIMIT $limit", clientColumns, p.scopeName, schemas.Collections.Client)
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
		sa := &schemas.Client{}
		if err := decodeDocument(raw, sa); err != nil {
			return nil, nil, err
		}
		clients = append(clients, sa)
	}
	if err := queryResult.Err(); err != nil {
		return nil, nil, err
	}
	return clients, paginationClone, nil
}
