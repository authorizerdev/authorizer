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

// AddClient creates a new service account record.
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.Client)
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

// UpdateClient updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct, per this
// method's "callers must load record first" contract.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.Client)
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

// DeleteClient removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	saCollection, _ := p.db.Collection(ctx, schemas.Collections.Client)
	_, err := saCollection.RemoveDocument(ctx, sa.Key)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("FOR d IN %s FILTER d.client_id == @client_id REMOVE { _key: d._key } IN %s", schemas.Collections.TrustedIssuer, schemas.Collections.TrustedIssuer)
	bindVars := map[string]interface{}{
		// TrustedIssuer.ServiceAccountID is stored as the bare key (it's set
		// verbatim from the external, API-facing id) — sa.ID is always the
		// full "collection/key" handle by the time this runs (populated from
		// the document's _id on every read). Compare against sa.Key, not sa.ID.
		"client_id": sa.Key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close() }()
	return nil
}

// GetClientByID fetches a service account by primary key.
// Filters on _key, not _id: every real caller (admin API params, the
// client_credentials token endpoint) holds the bare id AsAPIClient
// exposes, never the full "collection/key" handle.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	var sa *schemas.Client
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.Client)
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
		s := &schemas.Client{}
		if _, err := readDocument(ctx, cursor, s); err != nil {
			return nil, err
		}
		sa = s
	}
	return sa, nil
}

// ListClients returns a paginated list of service accounts.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	clients := []*schemas.Client{}
	query := fmt.Sprintf("FOR d in %s SORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.Client, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		sa := &schemas.Client{}
		meta, err := readDocument(ctx, cursor, sa)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			clients = append(clients, sa)
		}
	}
	return clients, paginationClone, nil
}
