package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddWebauthnCredential persists a newly registered passkey.
func (p *provider) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.CreatedAt = time.Now().Unix()
	cred.UpdatedAt = time.Now().Unix()
	return cred, nil
}

// UpdateWebauthnCredential writes back mutable fields (sign_count, flags,
// last_used_at, name). Caller must load the full record first.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	cred.UpdatedAt = time.Now().Unix()
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	return nil
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	return nil, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique
// WebAuthn credential id — the usernameless-login lookup.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	return nil, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	return nil, nil
}
