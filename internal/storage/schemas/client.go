package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Client is a registered OAuth client in the unified client registry.
//
// The Kind discriminator distinguishes interactive, human-facing clients
// (browser/app login flows) from machine service accounts (the OAuth2
// client_credentials grant and workload identity federation). Both kinds
// share this single registry so client authentication has one authoritative
// source of truth.
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
type Client struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// Kind distinguishes the client type. Values: "interactive" (human-facing
	// login clients) | "service_account" (machine/workload clients). It is
	// immutable after creation. This rename step defaults existing and newly
	// created rows to "service_account"; the interactive kind and its
	// additional fields (redirect_uris, grant_types, etc.) land in a later step.
	Kind string `json:"kind" bson:"kind" cql:"kind" dynamo:"kind"`

	// Name is a human-readable label for this service account (e.g. "payments-worker").
	Name string `json:"name" bson:"name" cql:"name" dynamo:"name"`

	// Description is an optional free-text note.
	Description *string `json:"description" bson:"description" cql:"description" dynamo:"description"`

	// ClientSecret holds the bcrypt hash of the credential.
	// Never expose this value in API responses.
	// json:"-" keeps it out of any json.Marshal of this struct (structured
	// logging, webhook payloads, error dumps), matching User.Password.
	ClientSecret string `json:"-" bson:"client_secret" cql:"client_secret" dynamo:"client_secret"`

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

// ParsedAllowedScopes returns AllowedScopes as a slice: comma-separated,
// whitespace trimmed, empty segments dropped. This is the single source of
// truth for interpreting the stored scope string — the token endpoint uses it
// to enforce that a client_credentials request stays within the authorized set.
// An empty or whitespace-only AllowedScopes yields an empty slice, which the
// token endpoint MUST treat as DENY-ALL (schema § AllowedScopes comment).
func (s *Client) ParsedAllowedScopes() []string {
	scopes := []string{}
	for _, sc := range strings.Split(s.AllowedScopes, ",") {
		if sc = strings.TrimSpace(sc); sc != "" {
			scopes = append(scopes, sc)
		}
	}
	return scopes
}

// AsAPIClient converts the storage record into the GraphQL model.
// It never exposes ClientSecret — there is no client_secret field on
// model.Client by design; the plaintext is surfaced only once via
// CreateClientResponse at creation/rotation.
func (s *Client) AsAPIClient() *model.Client {
	id := s.ID
	if strings.Contains(id, Collections.Client+"/") {
		id = strings.TrimPrefix(id, Collections.Client+"/")
	}
	return &model.Client{
		ID:            id,
		Name:          s.Name,
		Description:   s.Description,
		AllowedScopes: s.ParsedAllowedScopes(),
		IsActive:      s.IsActive,
		CreatedAt:     refs.NewInt64Ref(s.CreatedAt),
		UpdatedAt:     refs.NewInt64Ref(s.UpdatedAt),
	}
}
