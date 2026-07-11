package schemas

import (
	"errors"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// ErrOrgDomainConflict is returned by AddOrgDomain when the normalized domain
// is already verified by a DIFFERENT organization. It is the atomic
// first-writer-wins signal — the service layer maps it to
// "domain_already_verified_by_another_org". A same-org re-verify is NOT a
// conflict (AddOrgDomain returns the existing row, idempotently).
var ErrOrgDomainConflict = errors.New("domain already verified by another organization")

// OrgDomain is a VERIFIED mapping from a DNS domain to exactly one organization,
// used for home-realm discovery (routing a login to the right tenant IdP). A row
// exists ONLY once a domain is verified; a pending verification challenge is
// ephemeral state in the memory store, never a row here.
//
// Uniqueness is the load-bearing security invariant (one verified domain → one
// org). It is enforced by the PRIMARY/partition key being the normalized domain
// itself, NOT a secondary unique index: DynamoDB, Cassandra and Couchbase cannot
// enforce a unique constraint on a non-key attribute, so a check-then-insert
// guard would race. With the domain as the key, first-writer-wins is atomic on
// every backend (unique PK / conditional put / INSERT ... IF NOT EXISTS).
//
// ID holds the normalized domain and is the primary key. Domain holds the same
// value as a human-readable column (never expose ID directly — on ArangoDB a
// read populates it with the "collection/key" handle; use Domain / AsAPIOrgDomain).
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type OrgDomain struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	// ID is the primary key: the normalized domain (punycode, lowercased). Not a
	// uuid — the domain IS the key so uniqueness is enforced by the DB atomically.
	ID string `gorm:"primaryKey;type:varchar(255)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID is the organization this verified domain routes to. Indexed for
	// ListOrgDomainsByOrg; NOT unique (an org may verify many domains).
	OrgID string `gorm:"index" json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" index:"org_id,hash"`

	// Domain is the normalized domain as a readable column (== ID).
	Domain string `json:"domain" bson:"domain" cql:"domain" dynamo:"domain"`

	// VerifiedAt is when the domain became verified (unix seconds).
	VerifiedAt int64 `json:"verified_at" bson:"verified_at" cql:"verified_at" dynamo:"verified_at"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIOrgDomain converts the storage record into the GraphQL model. It exposes
// Domain (never ID — on ArangoDB a read populates ID with the "collection/key"
// handle).
func (d *OrgDomain) AsAPIOrgDomain() *model.OrgDomain {
	return &model.OrgDomain{
		Domain:     d.Domain,
		OrgID:      d.OrgID,
		VerifiedAt: refs.NewInt64Ref(d.VerifiedAt),
		CreatedAt:  refs.NewInt64Ref(d.CreatedAt),
		UpdatedAt:  refs.NewInt64Ref(d.UpdatedAt),
	}
}
