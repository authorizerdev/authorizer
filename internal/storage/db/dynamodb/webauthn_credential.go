package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebauthnCredential persists a newly registered passkey.
func (p *provider) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.Key = cred.ID
	now := time.Now().Unix()
	cred.CreatedAt = now
	cred.UpdatedAt = now
	if err := p.putItem(ctx, schemas.Collections.WebauthnCredential, cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// UpdateWebauthnCredential updates a passkey record.
// Callers MUST load the existing record and mutate it before calling this
// method — UpdateItem applies a partial SET/REMOVE merge that overwrites every
// supplied field, so a partial struct blanks untouched columns to their zero
// values.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.WebauthnCredential, "id", cred.ID, cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	if cred == nil {
		return nil
	}
	return p.deleteItemByHash(ctx, schemas.Collections.WebauthnCredential, "id", cred.ID)
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	var cred schemas.WebauthnCredential
	err := p.getItemByHash(ctx, schemas.Collections.WebauthnCredential, "id", id, &cred)
	if err != nil {
		return nil, err
	}
	if cred.ID == "" {
		return nil, errors.New("no document found")
	}
	return &cred, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique WebAuthn
// credential id — served by the credential_id GSI.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.WebauthnCredential, "credential_id", "credential_id", credentialID, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var cred schemas.WebauthnCredential
	if err := unmarshalItem(items[0], &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys, newest first.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	items, err := p.queryEq(ctx, schemas.Collections.WebauthnCredential, "user_id", "user_id", userID, nil)
	if err != nil {
		return nil, err
	}
	var creds []*schemas.WebauthnCredential
	for _, it := range items {
		var cred schemas.WebauthnCredential
		if err := unmarshalItem(it, &cred); err != nil {
			return nil, err
		}
		creds = append(creds, &cred)
	}
	sort.Slice(creds, func(i, j int) bool { return creds[i].CreatedAt > creds[j].CreatedAt })
	return creds, nil
}
