package schemas

// FederatedIdentity records that an end user was provisioned (JIT) from a
// specific upstream identity at a specific organization's SSO connection.
//
// SECURITY (design §4.4, account-takeover defense): a federated login is keyed
// by the tuple (OrgID, Issuer, Subject) — NEVER by email alone. On callback the
// broker looks up this row to find the already-provisioned Authorizer user for a
// returning federated principal; an email that merely collides with some other
// account is NOT sufficient to link. The (org_id, issuer, subject) triple is
// UNIQUE: the SQL providers enforce it with a composite unique index; the NoSQL
// providers enforce it with a compound unique index (Mongo/Arango) or a
// check-then-insert guard (Cassandra/Couchbase/DynamoDB). The service layer also
// looks up before inserting so behaviour is uniform across every backend.
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type FederatedIdentity struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID is the organization whose SSO connection provisioned the identity.
	// Part of the unique (org_id, issuer, subject) constraint.
	OrgID string `gorm:"uniqueIndex:idx_federated_identity_org_iss_sub" json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" index:"org_id,hash"`

	// Issuer is the upstream IdP's `iss` claim value.
	// Part of the unique (org_id, issuer, subject) constraint.
	Issuer string `gorm:"uniqueIndex:idx_federated_identity_org_iss_sub" json:"issuer" bson:"issuer" cql:"issuer" dynamo:"issuer"`

	// Subject is the upstream `sub` claim value (stable per-IdP user identifier).
	// Part of the unique (org_id, issuer, subject) constraint.
	Subject string `gorm:"uniqueIndex:idx_federated_identity_org_iss_sub" json:"subject" bson:"subject" cql:"subject" dynamo:"subject"`

	// UserID is the Authorizer user this federated identity maps to.
	UserID string `json:"user_id" bson:"user_id" cql:"user_id" dynamo:"user_id" gorm:"index" index:"user_id,hash"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
