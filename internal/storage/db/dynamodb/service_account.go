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

// AddServiceAccount creates a new service account record.
func (p *provider) AddServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	sa.Key = sa.ID
	now := time.Now().Unix()
	sa.CreatedAt = now
	sa.UpdatedAt = now
	if err := p.putItem(ctx, schemas.Collections.ServiceAccount, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// UpdateServiceAccount updates a service account record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge that overwrites every
// supplied field, so a partial struct blanks untouched columns to their zero
// values.
func (p *provider) UpdateServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) (*schemas.ServiceAccount, error) {
	if sa.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateServiceAccount: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sa.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.ServiceAccount, "id", sa.ID, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

// DeleteServiceAccount removes a service account and all its associated
// TrustedIssuers. Mirrors the webhook cascade-delete pattern.
func (p *provider) DeleteServiceAccount(ctx context.Context, sa *schemas.ServiceAccount) error {
	if sa == nil {
		return nil
	}
	// Delete child TrustedIssuers BEFORE the parent (mirrors the SQL cascade
	// ordering). A failure here leaves parent+children both present — a safe,
	// retryable state — rather than deleting the parent and orphaning issuers
	// that can still authenticate client_assertion JWTs. Any query/delete error
	// is returned so the caller knows the cascade did not complete.
	items, err := p.queryEq(ctx, schemas.Collections.TrustedIssuer, "service_account_id", "service_account_id", sa.ID, nil)
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
	return p.deleteItemByHash(ctx, schemas.Collections.ServiceAccount, "id", sa.ID)
}

// GetServiceAccountByID fetches a service account by primary key.
func (p *provider) GetServiceAccountByID(ctx context.Context, id string) (*schemas.ServiceAccount, error) {
	var sa schemas.ServiceAccount
	err := p.getItemByHash(ctx, schemas.Collections.ServiceAccount, "id", id, &sa)
	if err != nil {
		return nil, err
	}
	if sa.ID == "" {
		return nil, errors.New("no document found")
	}
	return &sa, nil
}

// ListServiceAccounts returns a paginated list of service accounts.
func (p *provider) ListServiceAccounts(ctx context.Context, pagination *model.Pagination) ([]*schemas.ServiceAccount, *model.Pagination, error) {
	paginationClone := pagination
	var serviceAccounts []*schemas.ServiceAccount

	items, err := p.scanAllRaw(ctx, schemas.Collections.ServiceAccount, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, it := range items {
		var sa schemas.ServiceAccount
		if err := unmarshalItem(it, &sa); err != nil {
			return nil, nil, err
		}
		serviceAccounts = append(serviceAccounts, &sa)
	}

	sort.Slice(serviceAccounts, func(i, j int) bool { return serviceAccounts[i].CreatedAt > serviceAccounts[j].CreatedAt })
	paginationClone.Total = int64(len(serviceAccounts))

	start := int(pagination.Offset)
	if start >= len(serviceAccounts) {
		return []*schemas.ServiceAccount{}, paginationClone, nil
	}
	end := start + int(pagination.Limit)
	if end > len(serviceAccounts) {
		end = len(serviceAccounts)
	}
	return serviceAccounts[start:end], paginationClone, nil
}
