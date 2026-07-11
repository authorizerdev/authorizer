package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// WebauthnCredential is a single WebAuthn/passkey credential registered to a
// User. One User may own many credentials (multiple devices/passkeys).
//
// Binary WebAuthn values (credential id, COSE public key, AAGUID) are stored as
// base64-standard strings and Transports as a comma-separated string so the row
// persists uniformly across all backends (SQL text, Cassandra text, DynamoDB S)
// — the same []byte/[]string-avoidance convention used by every other schema.
//
// Note: any field addition must also be reflected in the cassandradb provider.
type WebauthnCredential struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// UserID links the credential to its owning User. Indexed to list a user's
	// passkeys and to authorize deletion against the authenticated caller.
	UserID string `json:"user_id" bson:"user_id" cql:"user_id" dynamo:"user_id" gorm:"index" index:"user_id,hash"`

	// CredentialID is the base64-standard-encoded WebAuthn credential ID. It is
	// globally UNIQUE and is the only key that resolves a user during a
	// usernameless (discoverable) login, so it must be indexed and unique.
	CredentialID string `gorm:"uniqueIndex" json:"credential_id" bson:"credential_id" cql:"credential_id" dynamo:"credential_id" index:"credential_id,hash"`

	// PublicKey is the base64-standard-encoded COSE public key used to verify
	// assertions.
	PublicKey string `json:"public_key" bson:"public_key" cql:"public_key" dynamo:"public_key"`

	// SignCount is the authenticator signature counter. go-webauthn rejects a
	// login whose counter regresses below this value (cloned-authenticator
	// detection); it is written back on every successful login.
	SignCount int64 `json:"sign_count" bson:"sign_count" cql:"sign_count" dynamo:"sign_count"`

	// Flags is the raw single-byte go-webauthn CredentialFlags (UP/UV/BE/BS).
	// NOT in the original design spec but REQUIRED for correctness: go-webauthn's
	// login validation rejects a credential whose stored BackupEligible flag
	// disagrees with the assertion, and synced passkeys report BE=1 — so the flag
	// captured at registration must be persisted and restored, or every synced
	// passkey login fails.
	Flags int64 `json:"flags" bson:"flags" cql:"flags" dynamo:"flags"`

	// Transports is the comma-separated list of authenticator transports the
	// credential supports (e.g. "internal,hybrid").
	Transports string `json:"transports" bson:"transports" cql:"transports" dynamo:"transports"`

	// AAGUID is the base64-standard-encoded authenticator model identifier.
	AAGUID string `json:"aaguid" bson:"aaguid" cql:"aaguid" dynamo:"aaguid"`

	// Name is a user-supplied label (e.g. "MacBook Touch ID").
	Name string `json:"name" bson:"name" cql:"name" dynamo:"name"`

	CreatedAt  int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt  int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
	LastUsedAt *int64 `json:"last_used_at" bson:"last_used_at" cql:"last_used_at" dynamo:"last_used_at"`
}

// ParsedTransports returns Transports as a slice: comma-separated, trimmed,
// empty segments dropped.
func (w *WebauthnCredential) ParsedTransports() []string {
	transports := []string{}
	for _, t := range strings.Split(w.Transports, ",") {
		if t = strings.TrimSpace(t); t != "" {
			transports = append(transports, t)
		}
	}
	return transports
}

// AsAPIWebauthnCredential converts the storage record into the GraphQL model.
// It never exposes the public key, sign count or AAGUID — only the metadata a
// user needs to manage their passkeys.
func (w *WebauthnCredential) AsAPIWebauthnCredential() *model.WebauthnCredentialInfo {
	id := w.ID
	if strings.Contains(id, Collections.WebauthnCredential+"/") {
		id = strings.TrimPrefix(id, Collections.WebauthnCredential+"/")
	}
	return &model.WebauthnCredentialInfo{
		ID:         id,
		Name:       w.Name,
		Transports: w.ParsedTransports(),
		CreatedAt:  refs.NewInt64Ref(w.CreatedAt),
		UpdatedAt:  refs.NewInt64Ref(w.UpdatedAt),
		LastUsedAt: w.LastUsedAt,
	}
}
