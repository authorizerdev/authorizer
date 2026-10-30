package sql

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// skipUniqueIndexDrop reports whether AutoMigrate is about to drop a unique
// index the models declare, and should be left alone.
//
// GORM reconciles single-column uniqueness by comparing what the catalog
// reports against the field's `unique` tag. On a disagreement it issues
// DROP CONSTRAINT "uni_<table>_<column>" — a NamingStrategy-derived name that
// only ever exists for a constraint GORM itself created from a `unique` tag.
//
// PostgreSQL lists only real UNIQUE constraints in
// information_schema.table_constraints, so the two agree and nothing happens.
// CockroachDB lists UNIQUE *indexes* there too, and MySQL reports
// COLUMN_KEY='UNI' for them, so on both every `gorm:"uniqueIndex"` field looks
// like a constraint the model no longer wants — and startup dies on a name that
// was never created:
//
//	cockroachdb: constraint "uni_authorizer_oauth_states_state_key" of relation
//	             "authorizer_oauth_states" does not exist (SQLSTATE 42704)
//	mysql:       Error 1091 (42000): Can't DROP
//	             'uni_authorizer_oauth_states_state_key'
//
// Those indexes are declared by the models and are meant to be there, so the
// drop is skipped. The create direction — a `gorm:"unique"` field whose
// constraint is genuinely missing — is always delegated untouched.
//
// Each affected driver wraps its own concrete migrator (see cockroachdb.go and
// mysql.go) rather than the gorm.Migrator interface: gorm type-asserts
// migrator.BuildIndexOptionsInterface with no ok-check, so a wrapper that
// embeds the interface panics the moment a table with an index is created.
func skipUniqueIndexDrop(field *schema.Field, columnType gorm.ColumnType) bool {
	unique, ok := columnType.Unique()
	return ok && unique && !field.Unique && !field.PrimaryKey
}
