package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const webauthnCredentialColumns = "_id, user_id, credential_id, public_key, sign_count, flags, transports, aaguid, name, created_at, updated_at, last_used_at"

// AddWebauthnCredential persists a newly registered passkey.
//
// ponytail: check-then-insert has no atomic uniqueness guard on credential_id
// - Collection.Insert only enforces uniqueness on the document key (cred.ID),
// so two concurrent AddWebauthnCredential calls can both pass this pre-check.
// Matches the same accepted trade-off as AddUser's email/phone_number check
// on this backend; a real fix needs a Couchbase unique secondary index plus a
// conditional/index-backed insert. Real WebAuthn credential IDs are ≥16 bytes
// of random/key-wrapped authenticator data, so an actual collision here is
// cryptographically negligible - this exists for defense-in-depth.
func (p *provider) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CredentialID != "" {
		if existing, _ := p.GetWebauthnCredentialByCredentialID(ctx, cred.CredentialID); existing != nil {
			return nil, fmt.Errorf("a passkey with this credential id already exists")
		}
	}
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.Key = cred.ID
	now := time.Now().Unix()
	cred.CreatedAt = now
	cred.UpdatedAt = now
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(cred)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.WebauthnCredential).Insert(cred.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// UpdateWebauthnCredential updates a passkey record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	credMap, err := structToDocument(cred)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(credMap)
	params["_id"] = cred.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.WebauthnCredential, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.WebauthnCredential).Remove(cred.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, webauthnCredentialColumns, p.scopeName, schemas.Collections.WebauthnCredential)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	cred := &schemas.WebauthnCredential{}
	if err := decodeDocument(raw, cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique WebAuthn
// credential id — served by the credential_id index.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	params := make(map[string]interface{}, 1)
	params["credential_id"] = credentialID
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE credential_id=$credential_id LIMIT 1`, webauthnCredentialColumns, p.scopeName, schemas.Collections.WebauthnCredential)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	cred := &schemas.WebauthnCredential{}
	if err := decodeDocument(raw, cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys, newest first.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	creds := []*schemas.WebauthnCredential{}
	params := make(map[string]interface{}, 1)
	params["user_id"] = userID
	query := fmt.Sprintf("SELECT %s FROM %s.%s WHERE user_id=$user_id ORDER BY created_at DESC", webauthnCredentialColumns, p.scopeName, schemas.Collections.WebauthnCredential)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, err
		}
		cred := &schemas.WebauthnCredential{}
		if err := decodeDocument(raw, cred); err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	return creds, nil
}
