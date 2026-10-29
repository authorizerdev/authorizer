package sql

import (
	"fmt"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// sqlServerIndexedStringSize is the column width GORM's sqlserver driver
// already gives an unsized string that carries an `index` or `unique` tag.
const sqlServerIndexedStringSize = 256

// sqlServerDialector is the SQL Server dialector with its unsized-string
// sizing corrected. It embeds sqlserver.Dialector (which holds a *Config, so
// Initialize's writes are visible here) rather than the gorm.Dialector
// interface, so the driver's own methods all stay promoted.
//
// Unlike CockroachDB and MySQL, SQL Server needs no MigrateColumnUnique
// override: its ColumnTypes reads uniqueness from INFORMATION_SCHEMA.
// TABLE_CONSTRAINTS, which lists real UNIQUE constraints only — a
// CREATE UNIQUE INDEX is not a constraint there, so the catalog and the models
// agree, exactly as on PostgreSQL.
type sqlServerDialector struct {
	sqlserver.Dialector
}

// newSQLServerDialector returns the SQL Server dialector for dbURL with the
// sizing fix layered on.
func newSQLServerDialector(dbURL string) gorm.Dialector {
	return sqlServerDialector{*sqlserver.Open(dbURL).(*sqlserver.Dialector)}
}

// Migrator keeps the driver's own migrator and only repoints the dialector the
// migrator resolves column types through: left as the driver built it, every
// DDL type decision would route back to the unwrapped dialector and skip
// DataTypeOf below.
func (d sqlServerDialector) Migrator(db *gorm.DB) gorm.Migrator {
	m := d.Dialector.Migrator(db).(sqlserver.Migrator)
	// Spelled out: sqlserver.Migrator embeds migrator.Migrator, which carries
	// the Config whose Dialector drives DataTypeOf.
	m.Migrator.Config.Dialector = d
	return m
}

// DataTypeOf gives an unsized string column under a `uniqueIndex` a width, so
// SQL Server can actually index it.
//
// The driver sizes an unsized string to nvarchar(256) when the field carries
// `index` or `unique`, but it checks only those two tags — a field tagged
// `uniqueIndex` (named or composite) is missed and becomes nvarchar(MAX),
// which SQL Server refuses as an index key, so the server cannot create its
// schema at all:
//
//	mssql: Column 'client_id' in table 'authorizer_clients' is of a type that
//	is invalid for use as a key column in an index.
//
// Same width the driver picks for the tags it does recognise, so an indexed
// column is sized consistently however its index was declared.
func (d sqlServerDialector) DataTypeOf(field *schema.Field) string {
	if field.DataType == schema.String && field.Size == 0 &&
		field.TagSettings["UNIQUEINDEX"] != "" && d.DefaultStringSize == 0 {
		return fmt.Sprintf("nvarchar(%d)", sqlServerIndexedStringSize)
	}
	return d.Dialector.DataTypeOf(field)
}
