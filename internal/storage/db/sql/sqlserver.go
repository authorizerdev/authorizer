package sql

import (
	"fmt"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
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

// Initialize installs the driver, then swaps its create callback for one that
// honours an explicit ON CONFLICT target; see sqlServerCreate.
func (d sqlServerDialector) Initialize(db *gorm.DB) error {
	if err := d.Dialector.Initialize(db); err != nil {
		return err
	}
	return db.Callback().Create().Replace("gorm:create", sqlServerCreate)
}

// sqlServerCreate is the driver's create callback, except for an upsert that
// names its conflict columns.
//
// The driver turns every clause.OnConflict into a MERGE whose ON clause is
// built from the primary key alone — clause.OnConflict.Columns is ignored. An
// upsert keyed on a unique index therefore never matches the existing row (the
// new row has a fresh id), falls through to INSERT, and fails on the index:
//
//	mssql: Cannot insert duplicate key row in object
//	'dbo.authorizer_authenticators' with unique index
//	'idx_authenticator_user_id_method'. (2601)
//
// So AddAuthenticator and AddVerificationRequest, which upsert on
// (user_id, method) and (email, identifier), errored on SQL Server where every
// other SQL backend replaced the row. With Columns set this builds the MERGE
// on those columns instead. Everything else — plain inserts and
// primary-key upserts such as DoNothing — still goes to the driver untouched.
func sqlServerCreate(db *gorm.DB) {
	onConflict, ok := db.Statement.Clauses["ON CONFLICT"].Expression.(clause.OnConflict)
	if !ok || len(onConflict.Columns) == 0 || db.Error != nil ||
		db.Statement.Schema == nil || db.Statement.SQL.Len() > 0 {
		sqlserver.Create(db)
		return
	}

	if !db.Statement.Unscoped {
		for _, c := range db.Statement.Schema.CreateClauses {
			db.Statement.AddClause(c)
		}
	}
	values := callbacks.ConvertToCreateValues(db.Statement)
	// ConvertToCreateValues expands UpdateAll into DoUpdates; read it back.
	onConflict, _ = db.Statement.Clauses["ON CONFLICT"].Expression.(clause.OnConflict)

	inserted := make(map[string]bool, len(values.Columns))
	for _, column := range values.Columns {
		inserted[column.Name] = true
	}
	for _, column := range onConflict.Columns {
		if !inserted[column.Name] {
			_ = db.AddError(fmt.Errorf("sqlserver: ON CONFLICT column %q is not part of the INSERT", column.Name))
			return
		}
	}

	hasOutput := buildConflictTargetMerge(db, onConflict, values)
	if db.DryRun || db.Error != nil {
		return
	}
	if hasOutput {
		rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
		if db.AddError(err) == nil {
			defer rows.Close() //nolint:errcheck
			gorm.Scan(rows, db, gorm.ScanUpdate|gorm.ScanOnConflictDoNothing)
			if db.Statement.Result != nil {
				db.Statement.Result.RowsAffected = db.RowsAffected
			}
		}
		return
	}
	result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	if db.AddError(err) == nil {
		db.RowsAffected, _ = result.RowsAffected()
		if db.Statement.Result != nil {
			db.Statement.Result.Result = result
			db.Statement.Result.RowsAffected = db.RowsAffected
		}
	}
}

// buildConflictTargetMerge writes
//
//	MERGE INTO t USING (VALUES ...) AS excluded (cols)
//	ON t.c1 = excluded.c1 AND ...
//	[WHEN MATCHED THEN UPDATE SET ...]
//	WHEN NOT MATCHED THEN INSERT (cols) VALUES (excluded.cols) [OUTPUT ...];
//
// It follows the driver's sqlserver.MergeCreate, matching on onConflict.Columns
// instead of the primary key. HOLDLOCK makes the match-then-insert atomic: a
// plain MERGE lets two concurrent upserts of the same key both take the
// NOT MATCHED branch, and the loser fails on the unique index — the very race
// these upserts exist to close.
func buildConflictTargetMerge(db *gorm.DB, onConflict clause.OnConflict, values clause.Values) bool {
	stmt := db.Statement
	// The statement buffers into a strings.Builder, whose writes cannot fail.
	write := func(str string) { _, _ = stmt.WriteString(str) }
	writeByte := func(c byte) { _ = stmt.WriteByte(c) }
	write("MERGE INTO ")
	stmt.WriteQuoted(clause.Table{Name: clause.CurrentTable})
	write(" WITH (HOLDLOCK) USING (VALUES")
	for idx, value := range values.Values {
		if idx > 0 {
			writeByte(',')
		}
		writeByte('(')
		stmt.AddVar(stmt, value...)
		writeByte(')')
	}
	write(") AS excluded (")
	for idx, column := range values.Columns {
		if idx > 0 {
			writeByte(',')
		}
		stmt.WriteQuoted(column.Name)
	}
	write(") ON ")

	var on clause.Where
	for _, column := range onConflict.Columns {
		on.Exprs = append(on.Exprs, clause.Eq{
			Column: clause.Column{Table: stmt.Table, Name: column.Name},
			Value:  clause.Column{Table: "excluded", Name: column.Name},
		})
	}
	on.Build(stmt)

	if len(onConflict.DoUpdates) > 0 {
		write(" WHEN MATCHED THEN UPDATE SET ")
		onConflict.DoUpdates.Build(stmt)
	}

	// An auto-increment primary key is generated by the server, so it is left
	// out of the INSERT, as the driver does.
	insertable := make([]clause.Column, 0, len(values.Columns))
	for _, column := range values.Columns {
		if pk := stmt.Schema.PrioritizedPrimaryField; pk == nil || !pk.AutoIncrement || pk.DBName != column.Name {
			insertable = append(insertable, column)
		}
	}
	write(" WHEN NOT MATCHED THEN INSERT (")
	for idx, column := range insertable {
		if idx > 0 {
			writeByte(',')
		}
		stmt.WriteQuoted(column.Name)
	}
	write(") VALUES (")
	for idx, column := range insertable {
		if idx > 0 {
			writeByte(',')
		}
		stmt.WriteQuoted(clause.Column{Table: "excluded", Name: column.Name})
	}
	write(")")

	// Same OUTPUT the driver adds so database-generated defaults are read back
	// into the struct.
	hasOutput := false
	for _, field := range stmt.Schema.FieldsWithDefaultDBValue {
		if !field.Readable {
			continue
		}
		if hasOutput {
			write(", INSERTED.")
		} else {
			write(" OUTPUT INSERTED.")
			hasOutput = true
		}
		stmt.AddVar(stmt, clause.Column{Name: field.DBName})
	}
	write(";")
	return hasOutput
}
