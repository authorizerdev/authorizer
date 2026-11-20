package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddTrustedIssuer to save a trusted issuer in database
func (p *provider) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.ID == "" {
		issuer.ID = uuid.New().String()
	}
	issuer.Key = issuer.ID
	now := time.Now().Unix()
	issuer.CreatedAt = now
	issuer.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.TrustedIssuer, issuer.ID, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// UpdateTrustedIssuer to update a trusted issuer in database
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.TrustedIssuer, issuer.ID, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// DeleteTrustedIssuer to delete a trusted issuer from database
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	return p.deleteByID(ctx, schemas.Collections.TrustedIssuer, issuer.ID)
}

// GetTrustedIssuerByID to get a trusted issuer by its row ID
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	return getByID[schemas.TrustedIssuer](ctx, p, schemas.Collections.TrustedIssuer, id, "trusted issuer")
}

// GetTrustedIssuerByIssuerURL to get a trusted issuer by its issuer URL
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	return getWhere[schemas.TrustedIssuer](ctx, p, schemas.Collections.TrustedIssuer, "issuer_url = $v", "trusted issuer", map[string]any{"v": issuerURL})
}

// GetTrustedIssuerByOrgIDAndKind to get an organization's trusted issuer of a kind
func (p *provider) GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID, kind string) (*schemas.TrustedIssuer, error) {
	return getWhere[schemas.TrustedIssuer](ctx, p, schemas.Collections.TrustedIssuer, "org_id = $o AND kind = $k", "trusted issuer", map[string]any{"o": orgID, "k": kind})
}

// ListTrustedIssuers lists trusted issuers, optionally only a client's, newest first
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	where, vars := "", map[string]any{}
	if serviceAccountID != "" {
		where, vars["c"] = "client_id = $c", serviceAccountID
	}
	rows, pag, err := page[schemas.TrustedIssuer](ctx, p, schemas.Collections.TrustedIssuer, where, vars, pagination)
	if err != nil {
		return nil, nil, err
	}
	return ptrs(rows), pag, nil
}
