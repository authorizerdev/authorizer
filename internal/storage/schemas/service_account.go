package schemas

// ServiceAccount is the machine/workload identity primitive used for the
// OAuth2 client_credentials grant and workload identity federation.
//
// Authentication:
//   - client_id     = ID (UUID)
//   - client_secret = bcrypt hash (cost 12); plaintext returned ONCE at
//     creation and ONCE at rotation — never again.
//
// This struct intentionally omits email, phone, MFA, social-login, and
// session fields — those belong to User. Mixing human and machine identity
// in the same table is an anti-pattern explicitly avoided here.
// See: docs/specs/WORKLOAD_IDENTITY_PROGRAM.md §3.
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type ServiceAccount struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// Name is a human-readable label for this service account (e.g. "payments-worker").
	Name string `json:"name" bson:"name" cql:"name" dynamo:"name"`

	// Description is an optional free-text note.
	Description *string `json:"description" bson:"description" cql:"description" dynamo:"description"`

	// ClientSecret holds the bcrypt hash of the credential.
	// Never expose this value in API responses.
	ClientSecret string `json:"client_secret" bson:"client_secret" cql:"client_secret" dynamo:"client_secret"`

	// AllowedScopes is a comma-separated list of OAuth2 scopes this service
	// account may request. Scopes in a client_credentials request MUST be a
	// strict subset of this list.
	//
	// SECURITY: the service layer MUST trim whitespace and drop empty segments
	// when parsing this field (e.g. "read, write" → ["read","write"]).
	// Empty string MUST be treated as DENY-ALL, not grant-all, in the token
	// endpoint — an unparseable or empty AllowedScopes must reject the request.
	AllowedScopes string `json:"allowed_scopes" bson:"allowed_scopes" cql:"allowed_scopes" dynamo:"allowed_scopes"`

	// IsActive controls whether this service account may authenticate.
	// Flipping to false blocks new token issuance immediately; existing
	// tokens remain valid until their exp.
	//
	// GORM NOTE: gorm:"default:true" means db.Create with IsActive=false will
	// persist as true (Go zero-value triggers the column default). The service
	// layer must always set IsActive explicitly and use Save (not Create-only)
	// when creating a disabled account.
	IsActive bool `json:"is_active" bson:"is_active" cql:"is_active" dynamo:"is_active" gorm:"default:true"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
