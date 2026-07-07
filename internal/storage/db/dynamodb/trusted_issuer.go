package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddTrustedIssuer creates a new trusted issuer record.
func (p *provider) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.ID == "" {
		issuer.ID = uuid.New().String()
	}
	issuer.Key = issuer.ID
	now := time.Now().Unix()
	issuer.CreatedAt = now
	issuer.UpdatedAt = now
	if err := p.putItem(ctx, schemas.Collections.TrustedIssuer, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// UpdateTrustedIssuer updates a trusted issuer record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge that overwrites every
// supplied field, so a partial struct blanks untouched columns to their zero
// values.
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.TrustedIssuer, "id", issuer.ID, issuer); err != nil {
		return nil, err
	}
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer record.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	if issuer == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.TrustedIssuer, "id", issuer.ID)
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	var issuer schemas.TrustedIssuer
	err := p.getItemByHash(ctx, schemas.Collections.TrustedIssuer, "id", id, &issuer)
	if err != nil {
		return nil, err
	}
	if issuer.ID == "" {
		return nil, errors.New("no document found")
	}
	return &issuer, nil
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// Called on every client_assertion validation — served by the issuer_url GSI.
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.TrustedIssuer, "issuer_url", "issuer_url", issuerURL, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var issuer schemas.TrustedIssuer
	if err := unmarshalItem(items[0], &issuer); err != nil {
		return nil, err
	}
	return &issuer, nil
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	paginationClone := pagination
	var issuers []*schemas.TrustedIssuer

	var items []map[string]types.AttributeValue
	var err error
	if serviceAccountID != "" {
		items, err = p.queryEq(ctx, schemas.Collections.TrustedIssuer, "service_account_id", "service_account_id", serviceAccountID, nil)
	} else {
		items, err = p.scanAllRaw(ctx, schemas.Collections.TrustedIssuer, nil, nil)
	}
	if err != nil {
		return nil, nil, err
	}
	for _, it := range items {
		var issuer schemas.TrustedIssuer
		if err := unmarshalItem(it, &issuer); err != nil {
			return nil, nil, err
		}
		issuers = append(issuers, &issuer)
	}

	sort.Slice(issuers, func(i, j int) bool { return issuers[i].CreatedAt > issuers[j].CreatedAt })
	paginationClone.Total = int64(len(issuers))

	start := int(pagination.Offset)
	if start >= len(issuers) {
		return []*schemas.TrustedIssuer{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(issuers) {
		end = len(issuers)
	}
	return issuers[start:end], paginationClone, nil
}
