package sql

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// cockroachDialector is the Postgres dialector with one migration behaviour
// changed. It embeds postgres.Dialector rather than the gorm.Dialector
// interface so the driver's optional interfaces — SavePoint/RollbackTo, which
// gorm type-asserts for nested transactions — stay promoted.
type cockroachDialector struct {
	postgres.Dialector
}

// newCockroachDialector returns the Postgres dialector for dbURL with the
// CockroachDB migrator behaviour layered on.
func newCockroachDialector(dbURL string) gorm.Dialector {
	return cockroachDialector{postgres.Dialector{Config: &postgres.Config{DSN: dbURL}}}
}

func (d cockroachDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return cockroachMigrator{d.Dialector.Migrator(db)}
}

type cockroachMigrator struct {
	gorm.Migrator
}

// MigrateColumnUnique keeps AutoMigrate from dropping our unique indexes on
// CockroachDB.
//
// CockroachDB lists UNIQUE *indexes* in information_schema.table_constraints;
// PostgreSQL lists only real UNIQUE constraints there. GORM reads that extra
// row as "this column has a UNIQUE constraint the model no longer asks for" for
// every `gorm:"uniqueIndex"` field, and issues
// DROP CONSTRAINT "uni_<table>_<column>" — a NamingStrategy-derived name that
// never exists, because the object is an index with its own name. AutoMigrate
// then aborts with SQLSTATE 42704 and the server never starts:
//
//	constraint "uni_authorizer_oauth_states_state_key" of relation
//	"authorizer_oauth_states" does not exist (SQLSTATE 42704)
//
// Those indexes are declared by the models and are meant to be there, so the
// drop is skipped. The create direction (a `gorm:"unique"` field whose
// constraint is genuinely missing) is delegated untouched.
func (m cockroachMigrator) MigrateColumnUnique(value any, field *schema.Field, columnType gorm.ColumnType) error {
	if unique, ok := columnType.Unique(); ok && unique && !field.Unique && !field.PrimaryKey {
		return nil
	}
	return m.Migrator.MigrateColumnUnique(value, field, columnType)
}

// dropUniqueConstraint removes a single-column UNIQUE constraint on
// CockroachDB, where `ALTER TABLE ... DROP CONSTRAINT` — what
// Migrator.DropConstraint emits — is rejected outright:
//
//	unimplemented: cannot drop UNIQUE constraint "authorizer_users_email_key"
//	using ALTER TABLE DROP CONSTRAINT, use DROP INDEX CASCADE instead
//	(SQLSTATE 0A000)
//
// A UNIQUE constraint there is an index, so it has to go in those terms.
// Without this the legacy-uniqueness cleanup silently does nothing on an
// upgraded v1 database, which leaves UNIQUE on authorizer_otps.phone_number —
// where v2 writes the empty string for every email OTP, so the second one
// would be rejected.
//
// The identifiers are internal: table comes from a fixed list and constraint
// from this database's own information_schema, never from a request.
func dropUniqueConstraint(db *gorm.DB, table, constraint string) error {
	return db.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %q@%q CASCADE`, table, constraint)).Error
}
