package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddClient to save a client in database
func (p *provider) AddClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	if sa.ClientID == "" {
		sa.ClientID = sa.ID
	}
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.Client, sa.ID, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateClient to update a client in database
func (p *provider) UpdateClient(ctx context.Context, sa *schemas.Client) (*schemas.Client, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateClient: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.Client, sa.ID, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteClient deletes a client and the trusted issuers that belong to it,
// atomically.
func (p *provider) DeleteClient(ctx context.Context, sa *schemas.Client) error {
	return p.transaction(ctx, `DELETE type::table($itb) WHERE client_id = $id RETURN NONE;
		DELETE type::record($tb, $id) RETURN NONE;`,
		map[string]any{"itb": schemas.Collections.TrustedIssuer, "tb": schemas.Collections.Client, "id": sa.ID})
}

// GetClientByID to get a client by its row ID
func (p *provider) GetClientByID(ctx context.Context, id string) (*schemas.Client, error) {
	return getByID[schemas.Client](ctx, p, schemas.Collections.Client, id, "client")
}

// GetClientByClientID returns the client with the given OAuth client_id.
//
// It returns (nil, nil) when there is none: the one documented exception to
// the not-found contract, because its callers tell "absent" from
// "unavailable" by the error alone.
func (p *provider) GetClientByClientID(ctx context.Context, clientID string) (*schemas.Client, error) {
	client, err := getWhere[schemas.Client](ctx, p, schemas.Collections.Client, "client_id = $v", "client", map[string]any{"v": clientID})
	if IsNotFound(err) {
		return nil, nil
	}
	return client, err
}

// ListClients to list clients, newest first
func (p *provider) ListClients(ctx context.Context, pagination *model.Pagination) ([]*schemas.Client, *model.Pagination, error) {
	rows, pag, err := page[schemas.Client](ctx, p, schemas.Collections.Client, "", nil, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}
