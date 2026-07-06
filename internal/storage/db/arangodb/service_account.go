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
	meta, err := saCollection.CreateDocument(ctx, sa)
	if err != nil {
		return nil, err
	}
	sa.Key = meta.Key
	sa.ID = meta.ID.String()
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — the document replace writes every field.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateServiceAccount: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.ServiceAccount)
	meta, err := saCollection.UpdateDocument(ctx, sa.Key, sa)
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
		"service_account_id": sa.ID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}

// GetServiceAccountByID fetches a service account by primary key.
func (p *provider) GetServiceAccountByID(ctx context.Context, id string) (*schemas.ServiceAccount, error) {
	var sa *schemas.ServiceAccount
	query := fmt.Sprintf("FOR d in %s FILTER d._id == @id LIMIT 1 RETURN d", schemas.Collections.ServiceAccount)
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
		_, err := cursor.ReadDocument(ctx, &sa)
		if err != nil {
			return nil, err
		}
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
		var sa *schemas.ServiceAccount
		meta, err := cursor.ReadDocument(ctx, &sa)
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
