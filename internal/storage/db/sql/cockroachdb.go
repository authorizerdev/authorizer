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

// Migrator keeps the driver's own migrator — and with it every method gorm
// expects to find, including the BuildIndexOptions it type-asserts with no
// ok-check — and replaces only the unique reconciliation.
func (d cockroachDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return cockroachMigrator{d.Dialector.Migrator(db).(postgres.Migrator)}
}

type cockroachMigrator struct {
	postgres.Migrator
}

// MigrateColumnUnique leaves the unique indexes the models declare alone; see
// skipUniqueIndexDrop for why CockroachDB needs this and PostgreSQL does not.
func (m cockroachMigrator) MigrateColumnUnique(value any, field *schema.Field, columnType gorm.ColumnType) error {
	if skipUniqueIndexDrop(field, columnType) {
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
