package cassandradb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const webauthnCredentialColumns = "id, user_id, credential_id, public_key, sign_count, flags, transports, aaguid, name, created_at, updated_at, last_used_at"

// scanWebauthnCredential maps the webauthnCredentialColumns projection onto a struct.
func scanWebauthnCredential(scan func(...interface{}) error, cred *schemas.WebauthnCredential) error {
	return scan(&cred.ID, &cred.UserID, &cred.CredentialID, &cred.PublicKey, &cred.SignCount, &cred.Flags, &cred.Transports, &cred.AAGUID, &cred.Name, &cred.CreatedAt, &cred.UpdatedAt, &cred.LastUsedAt)
}

// AddWebauthnCredential persists a newly registered passkey.
func (p *provider) AddWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.Key = cred.ID
	now := time.Now().Unix()
	cred.CreatedAt = now
	cred.UpdatedAt = now
	// CredentialID is globally unique (gorm uniqueIndex on the SQL side) and
	// GetWebauthnCredentialByCredentialID — the usernameless-login hot path —
	// expects a single match. Cassandra has no cross-attribute unique constraint,
	// so guard with a check-then-insert mirroring AddTrustedIssuer's issuer_url pre-check.
	// ponytail: inherent TOCTOU race — two concurrent inserts of the same
	// credential_id can both pass this check. Cassandra offers no atomic IF NOT
	// EXISTS on a non-partition-key column, so this closes the common case only.
	if existing, _ := p.GetWebauthnCredentialByCredentialID(ctx, cred.CredentialID); existing != nil {
		return nil, fmt.Errorf("webauthn credential with %s credential_id already exists", cred.CredentialID)
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.WebauthnCredential, webauthnCredentialColumns)
	err := p.db.Query(insertQuery, cred.ID, cred.UserID, cred.CredentialID, cred.PublicKey, cred.SignCount, cred.Flags, cred.Transports, cred.AAGUID, cred.Name, cred.CreatedAt, cred.UpdatedAt, cred.LastUsedAt).Exec()
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// UpdateWebauthnCredential updates a passkey record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks columns it does not carry.
func (p *provider) UpdateWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) (*schemas.WebauthnCredential, error) {
	if cred.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateWebauthnCredential: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	cred.UpdatedAt = time.Now().Unix()
	// Column names are sourced from the `cql` struct tag (not json.Marshal, which
	// drops json:"-" fields — see buildCQLColumnMap).
	credMap := buildCQLColumnMap(cred)
	updateFields := ""
	var updateValues []interface{}
	for key, value := range credMap {
		if key == "id" {
			continue
		}
		if key == "_key" {
			continue
		}
		if value == nil {
			updateFields += fmt.Sprintf("%s = null,", key)
			continue
		}
		updateFields += fmt.Sprintf("%s = ?, ", key)
		updateValues = append(updateValues, value)
	}
	updateFields = strings.Trim(updateFields, " ")
	updateFields = strings.TrimSuffix(updateFields, ",")
	updateValues = append(updateValues, cred.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", KeySpace+"."+schemas.Collections.WebauthnCredential, updateFields)
	err := p.db.Query(query, updateValues...).Exec()
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// DeleteWebauthnCredential removes a passkey.
func (p *provider) DeleteWebauthnCredential(ctx context.Context, cred *schemas.WebauthnCredential) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", KeySpace+"."+schemas.Collections.WebauthnCredential)
	return p.db.Query(query, cred.ID).Exec()
}

// GetWebauthnCredentialByID fetches a passkey by primary key.
func (p *provider) GetWebauthnCredentialByID(ctx context.Context, id string) (*schemas.WebauthnCredential, error) {
	var cred schemas.WebauthnCredential
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ? LIMIT 1", webauthnCredentialColumns, KeySpace+"."+schemas.Collections.WebauthnCredential)
	err := scanWebauthnCredential(p.db.Query(query, id).Consistency(gocql.One).Scan, &cred)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// GetWebauthnCredentialByCredentialID resolves a passkey by its unique WebAuthn
// credential id — served by the credential_id secondary index.
func (p *provider) GetWebauthnCredentialByCredentialID(ctx context.Context, credentialID string) (*schemas.WebauthnCredential, error) {
	var cred schemas.WebauthnCredential
	query := fmt.Sprintf("SELECT %s FROM %s WHERE credential_id = ? LIMIT 1 ALLOW FILTERING", webauthnCredentialColumns, KeySpace+"."+schemas.Collections.WebauthnCredential)
	err := scanWebauthnCredential(p.db.Query(query, credentialID).Consistency(gocql.One).Scan, &cred)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// ListWebauthnCredentialsByUserID returns all of a user's passkeys, newest first.
func (p *provider) ListWebauthnCredentialsByUserID(ctx context.Context, userID string) ([]*schemas.WebauthnCredential, error) {
	creds := []*schemas.WebauthnCredential{}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE user_id = ? ALLOW FILTERING", webauthnCredentialColumns, KeySpace+"."+schemas.Collections.WebauthnCredential)
	scanner := p.db.Query(query, userID).Iter().Scanner()
	for scanner.Next() {
		var cred schemas.WebauthnCredential
		if err := scanWebauthnCredential(scanner.Scan, &cred); err != nil {
			return nil, err
		}
		creds = append(creds, &cred)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return creds, nil
}
