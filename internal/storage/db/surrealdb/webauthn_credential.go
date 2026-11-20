package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebauthnCredential to save a webauthn credential in database
func (p *provider) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.Key = cred.ID
	now := time.Now().Unix()
	cred.CreatedAt = now
	cred.UpdatedAt = now
	if err := p.create(ctx, schemas.Collections.WebauthnCredential, cred.ID, cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// UpdateWebauthnCredential to update a webauthn credential in database
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	if err := p.replace(ctx, schemas.Collections.WebauthnCredential, cred.ID, cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// DeleteWebauthnCredential to delete a webauthn credential from database
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	return p.deleteByID(ctx, schemas.Collections.WebauthnCredential, cred.ID)
}

// GetWebauthnCredentialByID to get a webauthn credential by its row ID
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	return getByID[schemas.WebauthnCredential](ctx, p, schemas.Collections.WebauthnCredential, id, "webauthn credential")
}

// GetWebauthnCredentialByCredentialID to get a webauthn credential by its credential ID
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	return getWhere[schemas.WebauthnCredential](ctx, p, schemas.Collections.WebauthnCredential, "credential_id = $v", "webauthn credential", map[string]any{"v": credentialID})
}

// ListWebauthnCredentialsByUserID to list a user's webauthn credentials, newest first
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	rows, err := listNewestFirst[schemas.WebauthnCredential](ctx, p, schemas.Collections.WebauthnCredential, "user_id = $u", map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	return ptrs(rows), nil
}
