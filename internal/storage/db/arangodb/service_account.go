package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddServiceAccount creates a new service account record.
func (p *provider) AddServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.ServiceAccount)
	doc, err := structToDocument(sa)
	if err != nil {
		return nil, err
	}
	meta, err := saCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	sa.Key = meta.Key
	sa.ID = meta.ID.String()
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct, per this
// method's "callers must load record first" contract.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateServiceAccount: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.ServiceAccount)
	doc, err := structToDocument(sa)
	if err != nil {
		return nil, err
	}
	meta, err := saCollection.UpdateDocument(ctx, sa.Key, doc)
	if err != nil {
		return nil, err
	}
	sa.Key = meta.Key
	sa.ID = meta.ID.String()
	return sa, nil
}

// DeleteServiceAccount removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) error {
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.ServiceAccount)
	_, err := saCollection.RemoveDocument(ctx, sa.Key)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("FOR d IN %s FILTER d.service_account_id == @service_account_id REMOVE { _key: d._key } IN %s", schemas.Collections.TrustedIssuer, schemas.Collections.TrustedIssuer)
	bindVars := map[string]interface{}{
		// TrustedIssuer.ServiceAccountID is stored as the bare key (it's set
		// verbatim from the external, API-facing id) — sa.ID is always the
		// full "collection/key" handle by the time this runs (populated from
		// the document's _id on every read). Compare against sa.Key, not sa.ID.
		"service_account_id": sa.Key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}

// GetServiceAccountByID fetches a service account by primary key.
// Filters on _key, not _id: every real caller (admin API params, the
// client_credentials token endpoint) holds the bare id AsAPIServiceAccount
// exposes, never the full "collection/key" handle.
func (p *provider) GetServiceAccountByID(ctx context.Context, id string) (*schemas.ServiceAccount, error) {
	var sa *schemas.ServiceAccount
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.ServiceAccount)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if sa == nil {
				return nil, fmt.Errorf("service account not found")
			}
			break
		}
		s := &schemas.ServiceAccount{}
		if _, err := readDocument(ctx, cursor, s); err != nil {
			return nil, err
		}
		sa = s
	}
	return sa, nil
}

// ListServiceAccounts returns a paginated list of service accounts.
func (p *provider) ListServiceAccounts(ctx context.Context, pagination *model.Pagination) ([]*schemas.ServiceAccount, *model.Pagination, error) {
	serviceAccounts := []*schemas.ServiceAccount{}
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.ServiceAccount, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		sa := &schemas.ServiceAccount{}
		meta, err := readDocument(ctx, cursor, sa)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			serviceAccounts = append(serviceAccounts, sa)
		}
	}
	return serviceAccounts, paginationClone, nil
}
