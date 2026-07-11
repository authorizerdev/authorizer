package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
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
	credCollection, _ := p.db.Collection(ctx, schemas.Collections.WebauthnCredential)
	doc, err := structToDocument(cred)
	if err != nil {
		return nil, err
	}
	meta, err := credCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	cred.Key = meta.Key
	cred.ID = meta.ID.String()
	return cred, nil
}

// UpdateWebauthnCredential updates a passkey record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct, per this
// method's "callers must load record first" contract.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	credCollection, _ := p.db.Collection(ctx, schemas.Collections.WebauthnCredential)
	doc, err := structToDocument(cred)
	if err != nil {
		return nil, err
	}
	meta, err := credCollection.UpdateDocument(ctx, cred.Key, doc)
	if err != nil {
		return nil, err
	}
	cred.Key = meta.Key
	cred.ID = meta.ID.String()
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	credCollection, _ := p.db.Collection(ctx, schemas.Collections.WebauthnCredential)
	_, err := credCollection.RemoveDocument(ctx, cred.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
// Filters on _key, not _id: every real caller holds the bare id, never the
// full "collection/key" handle.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	var cred *schemas.WebauthnCredential
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.WebauthnCredential)
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
			if cred == nil {
				return nil, fmt.Errorf("webauthn credential not found")
			}
			break
		}
		c := &schemas.WebauthnCredential{}
		if _, err := readDocument(ctx, cursor, c); err != nil {
			return nil, err
		}
		cred = c
	}
	return cred, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique WebAuthn
// credential id — the usernameless-login lookup.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	var cred *schemas.WebauthnCredential
	query := fmt.Sprintf("FOR d in %s FILTER d.credential_id == @credential_id LIMIT 1 RETURN d", schemas.Collections.WebauthnCredential)
	bindVars := map[string]interface{}{
		"credential_id": credentialID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if cred == nil {
				return nil, fmt.Errorf("webauthn credential not found")
			}
			break
		}
		c := &schemas.WebauthnCredential{}
		if _, err := readDocument(ctx, cursor, c); err != nil {
			return nil, err
		}
		cred = c
	}
	return cred, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys, newest first.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	creds := []*schemas.WebauthnCredential{}
	query := fmt.Sprintf("FOR d in %s FILTER d.user_id == @user_id SORT d.created_at DESC RETURN d", schemas.Collections.WebauthnCredential)
	bindVars := map[string]interface{}{
		"user_id": userID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		cred := &schemas.WebauthnCredential{}
		meta, err := readDocument(ctx, cursor, cred)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, err
		}
		if meta.Key != "" {
			creds = append(creds, cred)
		}
	}
	return creds, nil
}
