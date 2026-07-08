package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// ScimEndpoint is the per-organization inbound SCIM 2.0 connection credential.
// A customer's IdP (Okta, Entra, …) authenticates to /scim/v2/ with a bearer
// token; the org it may provision is derived ONLY from the matched endpoint
// (design §4.4 H6 — never from the URL or payload).
//
// Authentication:
//   - The presented bearer token is "<endpointID>.<hexSecret>". The endpointID
//     selects this row; the secret is verified (constant-time bcrypt) against
//     TokenHash. The plaintext is revealed ONCE at create and ONCE at rotate.
//
// One endpoint per org: OrgID is unique.
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type ScimEndpoint struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID is the organization this SCIM endpoint provisions into. Unique:
	// one SCIM endpoint per org.
	OrgID string `gorm:"uniqueIndex" json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" index:"org_id,hash"`

	// TokenHash is the bcrypt hash of the endpoint's bearer-token secret.
	// json:"-" keeps it out of any json.Marshal (structured logging, webhook
	// payloads, error dumps), matching User.Password / Client.ClientSecret.
	TokenHash string `json:"-" bson:"token_hash" cql:"token_hash" dynamo:"token_hash"`

	// Enabled gates whether the endpoint accepts requests. Disabling it stops
	// provisioning without deleting the row (rotate/re-enable later).
	//
	// GORM NOTE: gorm:"default:true" means Create with Enabled=false persists as
	// true (Go zero-value triggers the column default). The service layer always
	// sets Enabled explicitly and uses Save when creating a disabled endpoint.
	Enabled bool `gorm:"default:true" json:"enabled" bson:"enabled" cql:"enabled" dynamo:"enabled"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIScimEndpoint converts the storage record into the GraphQL model. The
// token secret is never included — it is revealed only at create/rotate.
func (s *ScimEndpoint) AsAPIScimEndpoint() *model.ScimEndpoint {
	id := s.ID
	if strings.Contains(id, Collections.ScimEndpoint+"/") {
		id = strings.TrimPrefix(id, Collections.ScimEndpoint+"/")
	}
	return &model.ScimEndpoint{
		ID:        id,
		OrgID:     s.OrgID,
		Enabled:   s.Enabled,
		CreatedAt: refs.NewInt64Ref(s.CreatedAt),
		UpdatedAt: refs.NewInt64Ref(s.UpdatedAt),
	}
}
