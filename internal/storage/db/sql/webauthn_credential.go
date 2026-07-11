package sql

import (
	"context"
	"fmt"
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
	res := p.db.Create(cred)
	if res.Error != nil {
		return nil, res.Error
	}
	return cred, nil
}

// UpdateWebauthnCredential writes back the record. Callers MUST load the
// existing record and mutate it before calling this method — Save writes every
// column and will blank zero-value fields on a partial struct.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	res := p.db.Save(cred)
	if res.Error != nil {
		return nil, res.Error
	}
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	return p.db.Delete(cred).Error
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	var cred schemas.WebauthnCredential
	res := p.db.Where("id = ?", id).First(&cred)
	if res.Error != nil {
		return nil, res.Error
	}
	return &cred, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique WebAuthn
// credential id — the usernameless-login lookup.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	var cred schemas.WebauthnCredential
	res := p.db.Where("credential_id = ?", credentialID).First(&cred)
	if res.Error != nil {
		return nil, res.Error
	}
	return &cred, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	var creds []*schemas.WebauthnCredential
	res := p.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&creds)
	if res.Error != nil {
		return nil, res.Error
	}
	return creds, nil
}
