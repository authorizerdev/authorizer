package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

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
	if sa.ClientID == "" {
		sa.ClientID = sa.ID
	}
	// DynamoDB has no unique constraint on a GSI, so guard client_id uniqueness
	// with a check-then-insert.
	// ponytail: inherent TOCTOU race — two concurrent inserts of the same
	// client_id can both pass this check; DynamoDB offers no cross-item
	// uniqueness. This closes the common case (sequential admin/boot-seed) only.
	if existing, _ := p.GetClientByClientID(ctx, sa.ClientID); existing != nil {
		return nil, fmt.Errorf("client with client_id %s already exists", sa.ClientID)
	}
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	if err := p.putItem(ctx, schemas.Collections.Client, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateClient updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge that overwrites every
// supplied field, so a partial struct blanks untouched columns to their zero
// values.
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Client, "id", sa.ID, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteClient removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	if sa == nil {
		return nil
	}
	// Delete child TrustedIssuers BEFORE the parent (mirrors the SQL cascade
	// ordering). A failure here leaves parent+children both present — a safe,
	// retryable state — rather than deleting the parent and orphaning issuers
	// that can still authenticate client_assertion JWTs. Any query/delete error
	// is returned so the caller knows the cascade did not complete.
	items, err := p.queryEq(ctx, schemas.Collections.TrustedIssuer, "client_id", "client_id", sa.ID, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var issuer schemas.TrustedIssuer
		if err := unmarshalItem(it, &issuer); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.TrustedIssuer, "id", issuer.ID); err != nil {
			return err
		}
	}
	return p.deleteItemByHash(ctx, schemas.Collections.Client, "id", sa.ID)
}

// GetClientByID fetches a service account by primary key.
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	var sa schemas.Client
	err := p.getItemByHash(ctx, schemas.Collections.Client, "id", id, &sa)
	if err != nil {
		return nil, err
	}
	if sa.ID == "" {
		return nil, errors.New("no document found")
	}
	return &sa, nil
}

// GetClientByClientID fetches a client by its unique public client_id.
// Served by the client_id GSI.
func (p *provider) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.Client, "client_id", "client_id", clientID, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		// No matching item is a normal negative result, not a storage failure —
		// callers (e.g. clientauth.ResolveClient) distinguish "no such client"
		// from "couldn't check" by whether err is nil.
		return nil, nil
	}
	var sa schemas.Client
	if err := unmarshalItem(items[0], &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

// ListClients returns a paginated list of service accounts.
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	paginationClone := pagination
	var clients []*schemas.Client

	items, err := p.scanAllRaw(ctx, schemas.Collections.Client, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, it := range items {
		var sa schemas.Client
		if err := unmarshalItem(it, &sa); err != nil {
			return nil, nil, err
		}
		clients = append(clients, &sa)
	}

	sort.Slice(clients, func(i, j int) bool { return clients[i].CreatedAt > clients[j].CreatedAt })
	paginationClone.Total = int64(len(clients))

	start := int(pagination.Offset)
	if start >= len(clients) {
		return []*schemas.Client{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(clients) {
		end = len(clients)
	}
	return clients[start:end], paginationClone, nil
}
