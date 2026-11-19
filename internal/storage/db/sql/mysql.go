package sql

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// indexedStringSize is the column width GORM's mysql driver already gives an
// unsized string that carries an `index` or `unique` tag: 191 characters, the
// widest utf8mb4 column InnoDB can index under the 767-byte prefix limit.
const indexedStringSize = 191

// mysqlDialector is the MySQL dialector with two migration behaviours changed.
// It embeds mysql.Dialector (which itself holds a *Config, so Initialize's
// writes are visible here) rather than the gorm.Dialector interface, so the
// driver's methods all stay promoted.
type mysqlDialector struct {
	mysql.Dialector
}

// newMySQLDialector returns the MySQL dialector for dbURL with the MySQL
// migration fixes layered on. It goes through mysql.New so the driver fills in
// DSNConfig from the DSN, as Initialize expects.
func newMySQLDialector(dbURL string) gorm.Dialector {
	return mysqlDialector{*mysql.New(mysql.Config{DSN: dbURL}).(*mysql.Dialector)}
}

// Migrator keeps the driver's own migrator — and with it every method gorm
// expects to find, including the BuildIndexOptions it type-asserts with no
// ok-check — and replaces only the unique reconciliation.
//
// Config.Dialector is repointed at this wrapper because
// migrator.Migrator.DataTypeOf resolves column types through it: left as the
// driver built it, every DDL type decision would route back to the unwrapped
// dialector and skip DataTypeOf below.
func (d mysqlDialector) Migrator(db *gorm.DB) gorm.Migrator {
	m := d.Dialector.Migrator(db).(mysql.Migrator)
	// Spelled out: mysql.Migrator embeds both migrator.Migrator and the
	// driver's Dialector, and each carries a Config, so a bare m.Config is
	// ambiguous.
	m.Migrator.Config.Dialector = d
	return mysqlMigrator{m}
}

type mysqlMigrator struct {
	mysql.Migrator
}

// MigrateColumnUnique leaves the unique indexes the models declare alone; see
// skipUniqueIndexDrop.
func (m mysqlMigrator) MigrateColumnUnique(value any, field *schema.Field, columnType gorm.ColumnType) error {
	if skipUniqueIndexDrop(field, columnType) {
		return nil
	}
	return m.Migrator.MigrateColumnUnique(value, field, columnType)
}

// DataTypeOf gives an unsized string column under a `uniqueIndex` a width, so
// MySQL can actually index it.
//
// The driver sizes an unsized string to varchar(191) when the field carries
// `index` or `unique`, but it checks only those two tags — a field tagged
// `uniqueIndex` (named or composite) is missed and becomes `longtext`. MySQL
// then rejects the table outright, because a TEXT column needs an explicit key
// length to be indexed, and the server cannot create its schema at all:
//
//	Error 1170 (42000): BLOB/TEXT column 'client_id' used in key specification
//	without a key length
//
// Same width the driver picks for the tags it does recognise, so an indexed
// column is sized consistently however its index was declared.
func (d mysqlDialector) DataTypeOf(field *schema.Field) string {
	if field.DataType == schema.String && field.Size == 0 &&
		field.TagSettings["UNIQUEINDEX"] != "" && d.DefaultStringSize == 0 {
		return fmt.Sprintf("varchar(%d)", indexedStringSize)
	}
	return d.Dialector.DataTypeOf(field)
}
