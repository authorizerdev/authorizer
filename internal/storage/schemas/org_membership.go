package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// OrgMembership links a User to an Organization with a set of per-org roles.
//
// A user may belong to many organizations, each with independent roles; an
// organization has many members. The (OrgID, UserID) pair is UNIQUE — a user
// can be a member of a given organization at most once. The SQL providers
// enforce this with a composite unique index; the NoSQL providers enforce it
// with a compound unique index (Mongo/Arango) or a check-then-insert guard
// (Cassandra/Couchbase/DynamoDB). The service layer additionally rejects
// duplicates up-front so behaviour is uniform across every backend.
//
// Roles is a comma-separated list of per-organization role names, mirroring the
// storage convention used by Client.AllowedScopes. Use ParsedRoles to interpret
// it; the service layer normalizes on write.
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type OrgMembership struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID references the Organization this membership belongs to.
	// Part of the unique (org_id, user_id) constraint.
	OrgID string `gorm:"uniqueIndex:idx_org_membership_org_user" json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" index:"org_id,hash"`

	// UserID references the User who is a member.
	// Part of the unique (org_id, user_id) constraint; also indexed on its own
	// for the "list an org membership by user" query.
	UserID string `gorm:"uniqueIndex:idx_org_membership_org_user;index" json:"user_id" bson:"user_id" cql:"user_id" dynamo:"user_id" index:"user_id,hash"`

	// Roles is a comma-separated list of per-organization roles.
	Roles string `json:"roles" bson:"roles" cql:"roles" dynamo:"roles"`

	// DeprovisionedAt marks this membership as deprovisioned by SCIM, in unix
	// seconds. nil means the membership is active.
	//
	// It is a nullable timestamp rather than an `IsActive bool` on purpose. A
	// bool's zero value is false, and none of the six backends can backfill a
	// new column for rows that already exist: SQL AutoMigrate adds it as NULL,
	// Cassandra's ALTER TABLE leaves it unset, and the document stores simply
	// have no such key. Every one of those decodes into Go as false, which with
	// an `IsActive` field would read as "deprovisioned" and lock every existing
	// member out of their organization on upgrade. Absent therefore has to mean
	// active, and it mirrors User.RevokedTimestamp, which already works this way.
	DeprovisionedAt *int64 `json:"deprovisioned_at" bson:"deprovisioned_at" cql:"deprovisioned_at" dynamo:"deprovisioned_at"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// Active reports whether this membership grants access to its organization.
//
// Every access-control path that resolves a membership must gate on this, not
// merely on the membership existing: a deprovisioned row is kept so that SCIM
// can report `active: false` and later reactivate it. Management paths (SCIM
// GET/PATCH/PUT) deliberately still see deprovisioned rows.
func (m *OrgMembership) Active() bool {
	return m != nil && m.DeprovisionedAt == nil
}

// ParsedRoles returns Roles as a slice: comma-separated, whitespace trimmed,
// empty segments dropped. An empty or whitespace-only Roles yields an empty
// slice.
func (m *OrgMembership) ParsedRoles() []string {
	roles := []string{}
	for _, r := range strings.Split(m.Roles, ",") {
		if r = strings.TrimSpace(r); r != "" {
			roles = append(roles, r)
		}
	}
	return roles
}

// AsAPIOrgMember converts the storage record into the GraphQL model.
func (m *OrgMembership) AsAPIOrgMember() *model.OrgMember {
	id := m.ID
	if strings.Contains(id, Collections.OrgMembership+"/") {
		id = strings.TrimPrefix(id, Collections.OrgMembership+"/")
	}
	return &model.OrgMember{
		ID:        id,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		Roles:     m.ParsedRoles(),
		CreatedAt: refs.NewInt64Ref(m.CreatedAt),
		UpdatedAt: refs.NewInt64Ref(m.UpdatedAt),
	}
}
