package schemas

// ScimGroup is a per-organization SCIM 2.0 Group resource (RFC 7643 §4.2). It
// carries ONLY the group's identity/metadata — displayName, org namespace,
// externalId, timestamps. Membership is deliberately NOT a column: per RFC 7643
// §4.1.2 membership is mutated through the Group resource (PATCH members) and is
// modelled as OpenFGA relationship tuples (group:<org>/<id>#member@user:<uid>),
// never as a stored list on the row. This keeps the group→role grant, nested
// groups, and the SAML group projection all resolvable through the one FGA graph.
//
// Org isolation: OrgID namespaces the group. A group is visible/mutable through a
// SCIM connection only for the org the bearer token resolved to (design §4.4 H6,
// mirrors ScimEndpoint). DisplayName uniqueness within an org is enforced by the
// service layer (GetScimGroupByOrgAndDisplayName pre-check), not a DB constraint,
// so behaviour is identical across all providers.
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type ScimGroup struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID is the organization this group belongs to. Indexed (NOT unique): an
	// org has many groups.
	OrgID string `gorm:"index" json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" index:"org_id,hash"`

	// DisplayName is the SCIM displayName (RFC 7643 §4.2). Unique within an org
	// (service-enforced).
	DisplayName string `json:"display_name" bson:"display_name" cql:"display_name" dynamo:"display_name"`

	// ExternalID is the IdP-side identifier, stored org-namespaced ("<orgID>:<raw>")
	// exactly like User.ExternalID so one org's connection can never resolve
	// another org's group by external id (H6). Nullable — some IdPs omit it.
	ExternalID *string `json:"external_id" bson:"external_id" cql:"external_id" dynamo:"external_id"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
