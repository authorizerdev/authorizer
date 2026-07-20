package schemas

// SAMLIDPKey is a per-organization SAML IdP signing keypair: an RSA private key
// and its self-signed X.509 certificate. Authorizer signs the SAML assertions it
// issues (as an IdP) with this key; downstream SPs pin the certificate.
//
// The JWT signing key (Config.JWTPrivateKey) is deliberately NOT reused: SAML
// XML-DSIG requires an X.509 certificate wrapper, which the raw-PEM JWT key does
// not carry, and SAML key rotation is independent of JWT key rotation.
//
// ROTATION (Status field):
//   - "current" — the single active key NEW assertions are signed with. There is
//     at most one "current" key per org.
//   - "active"  — a formerly-current key that is STILL published in IdP metadata
//     so that SPs which have not refreshed their cached metadata continue to
//     accept assertions. It is never used to sign new assertions.
//   - "retired" — explicitly retired by an admin; no longer published, no longer
//     usable. Retirement is an explicit admin action, never time-based.
//
// Metadata publishes every "current" + "active" key as separate <KeyDescriptor>
// signing entries. Signing always uses the "current" key only.
//
// Note: any field addition must also be reflected in the cassandradb provider.
type SAMLIDPKey struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID scopes this signing key to one Organization. Immutable.
	OrgID string `json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" gorm:"index" index:"org_id,hash"`

	// CertPEM is the self-signed X.509 certificate (PEM) wrapping the public key.
	// Published in IdP metadata; safe to expose.
	CertPEM string `json:"cert_pem" bson:"cert_pem" cql:"cert_pem" dynamo:"cert_pem"`

	// PrivateKeyEnc is the RSA private key (PEM), AES-encrypted at rest with
	// crypto.EncryptAES keyed on Config.ClientSecret — reversible because the raw
	// key must be reconstructed to sign. json:"-" so it NEVER serializes into an
	// API/webhook/log projection.
	PrivateKeyEnc string `json:"-" bson:"private_key_enc" cql:"private_key_enc" dynamo:"private_key_enc"`

	// Algorithm is the JWS-style signing algorithm identifier (e.g. "RS256"),
	// mapped to the XML-DSIG signature method at signing time.
	Algorithm string `json:"algorithm" bson:"algorithm" cql:"algorithm" dynamo:"algorithm"`

	// Status is one of "current", "active", "retired" (see type docs).
	Status string `json:"status" bson:"status" cql:"status" dynamo:"status" gorm:"default:'current'"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// SAML IdP key rotation statuses.
const (
	SAMLIDPKeyStatusCurrent = "current"
	SAMLIDPKeyStatusActive  = "active"
	SAMLIDPKeyStatusRetired = "retired"
)
