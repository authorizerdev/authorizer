package serenedb

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"

	"github.com/authorizerdev/authorizer/internal/storage/db/sql"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// These tests need no database: they run in `make test`, so a model change
// that is missing from the hand-written DDL fails CI rather than a SereneDB
// deployment.

type ddlColumn struct {
	typ, def string
}

type ddlIndex struct {
	unique bool
	cols   []string
}

type ddlTable struct {
	columns map[string]ddlColumn
	pk      []string
}

// ddlSchema is the schema the migrations produce, built by reading their DDL.
type ddlSchema struct {
	tables  map[string]*ddlTable
	indexes map[string]ddlIndex // by name; inline UNIQUE constraints included
	owner   map[string]string   // index name -> table
}

var (
	reCreateTable = regexp.MustCompile(`(?is)^CREATE TABLE IF NOT EXISTS (\w+) \((.*)\)$`)
	reCreateIndex = regexp.MustCompile(`(?is)^CREATE (UNIQUE )?INDEX IF NOT EXISTS (\w+) ON (\w+) \(([^)]*)\)$`)
	reAddColumn   = regexp.MustCompile(`(?is)^ALTER TABLE (\w+) ADD COLUMN IF NOT EXISTS (.+)$`)
	reDropColumn  = regexp.MustCompile(`(?is)^ALTER TABLE (\w+) DROP COLUMN IF EXISTS (\w+)$`)
	reDropIndex   = regexp.MustCompile(`(?is)^DROP INDEX IF EXISTS (\w+)$`)
	rePrimaryKey  = regexp.MustCompile(`(?is)^PRIMARY KEY \(([^)]*)\)$`)
	reUniqueCons  = regexp.MustCompile(`(?is)^CONSTRAINT (\w+) UNIQUE \(([^)]*)\)$`)
	reColumn      = regexp.MustCompile(`(?is)^(\w+) (.+?)(?: DEFAULT (.+))?$`)
)

func splitList(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if rest := strings.TrimSpace(s[start:]); rest != "" {
		out = append(out, rest)
	}
	return out
}

func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

func parseColumn(def string) (string, ddlColumn, error) {
	m := reColumn.FindStringSubmatch(def)
	if m == nil {
		return "", ddlColumn{}, fmt.Errorf("unparseable column %q", def)
	}
	return m[1], ddlColumn{typ: strings.ToLower(m[2]), def: m[3]}, nil
}

// parseMigrations replays list's DDL into a ddlSchema. It understands exactly
// the statement shapes schema.go is allowed to use; anything else is an error,
// so a new kind of statement has to be taught here before it can ship.
func parseMigrations(list []migration) (*ddlSchema, error) {
	s := &ddlSchema{tables: map[string]*ddlTable{}, indexes: map[string]ddlIndex{}, owner: map[string]string{}}
	addIndex := func(name, table string, idx ddlIndex) error {
		if _, dup := s.indexes[name]; dup {
			return fmt.Errorf("index %s declared twice", name)
		}
		s.indexes[name], s.owner[name] = idx, table
		return nil
	}
	for _, m := range list {
		for _, raw := range m.statements {
			stmt := normalizeSpace(raw)
			switch {
			case reCreateTable.MatchString(stmt):
				g := reCreateTable.FindStringSubmatch(stmt)
				if _, dup := s.tables[g[1]]; dup {
					return nil, fmt.Errorf("table %s created twice", g[1])
				}
				t := &ddlTable{columns: map[string]ddlColumn{}}
				s.tables[g[1]] = t
				for _, item := range splitList(g[2]) {
					switch {
					case rePrimaryKey.MatchString(item):
						t.pk = splitList(rePrimaryKey.FindStringSubmatch(item)[1])
					case reUniqueCons.MatchString(item):
						u := reUniqueCons.FindStringSubmatch(item)
						if err := addIndex(u[1], g[1], ddlIndex{unique: true, cols: splitList(u[2])}); err != nil {
							return nil, err
						}
					default:
						name, col, err := parseColumn(item)
						if err != nil {
							return nil, fmt.Errorf("table %s: %w", g[1], err)
						}
						t.columns[name] = col
					}
				}
			case reCreateIndex.MatchString(stmt):
				g := reCreateIndex.FindStringSubmatch(stmt)
				if err := addIndex(g[2], g[3], ddlIndex{unique: g[1] != "", cols: splitList(g[4])}); err != nil {
					return nil, err
				}
			case reAddColumn.MatchString(stmt):
				g := reAddColumn.FindStringSubmatch(stmt)
				t, ok := s.tables[g[1]]
				if !ok {
					return nil, fmt.Errorf("ADD COLUMN on unknown table %s", g[1])
				}
				name, col, err := parseColumn(g[2])
				if err != nil {
					return nil, err
				}
				t.columns[name] = col
			case reDropColumn.MatchString(stmt):
				g := reDropColumn.FindStringSubmatch(stmt)
				if t, ok := s.tables[g[1]]; ok {
					delete(t.columns, g[2])
				}
			case reDropIndex.MatchString(stmt):
				name := reDropIndex.FindStringSubmatch(stmt)[1]
				delete(s.indexes, name)
				delete(s.owner, name)
			default:
				return nil, fmt.Errorf("migration %d: statement shape not understood by the parity test "+
					"(extend parseMigrations to support it): %s", m.version, stmt)
			}
		}
	}
	return s, nil
}

// expectedType is the SereneDB column type for a model field: what GORM's
// Postgres dialect would declare, with every string type collapsed to text
// (see schema.go).
func expectedType(f *schema.Field) string {
	// An explicit `type:` tag (char(36), varchar(64), ...) arrives as the
	// DataType itself.
	if dt := strings.ToLower(string(f.DataType)); strings.HasPrefix(dt, "char") ||
		strings.HasPrefix(dt, "varchar") || dt == "text" {
		return "text"
	}
	switch f.DataType {
	case schema.String:
		return "text"
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		if f.Size <= 16 && f.Size > 0 {
			return "smallint"
		}
		if f.Size <= 32 && f.Size > 0 {
			return "integer"
		}
		return "bigint"
	case schema.Float:
		if f.Size <= 32 && f.Size > 0 {
			return "real"
		}
		return "double precision"
	case schema.Time:
		return "timestamptz"
	case schema.Bytes:
		return "bytea"
	}
	return fmt.Sprintf("<unmapped %s>", f.DataType)
}

func unquote(s string) string { return strings.Trim(strings.TrimSpace(s), "'") }

func parseModel(t *testing.T, model any) *schema.Schema {
	t.Helper()
	s, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{TablePrefix: schemas.Prefix})
	require.NoError(t, err)
	return s
}

func TestSchemaDDLMatchesModels(t *testing.T) {
	ddl, err := parseMigrations(migrations)
	require.NoError(t, err)

	wantTables := map[string]bool{}
	wantIndexes := map[string]bool{}
	for _, model := range sql.Models() {
		ms := parseModel(t, model)
		wantTables[ms.Table] = true
		t.Run(ms.Table, func(t *testing.T) {
			tbl, ok := ddl.tables[ms.Table]
			require.True(t, ok, "no CREATE TABLE for %s in schema.go", ms.Table)

			var wantPK []string
			wantCols := map[string]bool{}
			for _, f := range ms.Fields {
				if f.DBName == "" || f.IgnoreMigration {
					continue
				}
				wantCols[f.DBName] = true
				if f.PrimaryKey {
					wantPK = append(wantPK, f.DBName)
				}
				col, ok := tbl.columns[f.DBName]
				if !assert.True(t, ok, "column %s.%s (field %s) is missing from schema.go — add a migration", ms.Table, f.DBName, f.Name) {
					continue
				}
				assert.Equal(t, expectedType(f), col.typ, "type of %s.%s", ms.Table, f.DBName)
				assert.Equal(t, unquote(f.DefaultValue), unquote(col.def),
					"DEFAULT of %s.%s must match the model tag: GORM leaves zero-valued defaulted fields out of INSERTs", ms.Table, f.DBName)
			}
			for name := range tbl.columns {
				assert.True(t, wantCols[name], "schema.go declares %s.%s, which no model field maps to", ms.Table, name)
			}
			assert.Equal(t, wantPK, tbl.pk, "primary key of %s", ms.Table)

			// Every index and unique constraint GORM would have created.
			want := map[string]ddlIndex{}
			for _, idx := range ms.ParseIndexes() {
				cols := make([]string, 0, len(idx.Fields))
				for _, f := range idx.Fields {
					cols = append(cols, f.DBName)
				}
				want[idx.Name] = ddlIndex{unique: idx.Class == "UNIQUE", cols: cols}
			}
			for _, f := range ms.Fields {
				if f.Unique && !f.PrimaryKey {
					want[schema.NamingStrategy{}.UniqueName(ms.Table, f.DBName)] = ddlIndex{unique: true, cols: []string{f.DBName}}
				}
			}
			for name, w := range want {
				wantIndexes[name] = true
				got, ok := ddl.indexes[name]
				if !assert.True(t, ok, "index %s on %s is missing from schema.go", name, ms.Table) {
					continue
				}
				assert.Equal(t, ms.Table, ddl.owner[name], "index %s is on the wrong table", name)
				assert.Equal(t, w.unique, got.unique, "uniqueness of %s", name)
				assert.Equal(t, w.cols, got.cols, "columns of %s", name)
			}
		})
	}
	for name := range ddl.tables {
		assert.True(t, wantTables[name], "schema.go creates %s, which is not in sql.Models()", name)
	}
	for name := range ddl.indexes {
		assert.True(t, wantIndexes[name], "schema.go declares index %s, which no model declares", name)
	}
}

// The parity test is only as good as its parser: prove it notices drift.
func TestSchemaDDLParityDetectsDrift(t *testing.T) {
	base, err := parseMigrations(migrations)
	require.NoError(t, err)
	require.Contains(t, base.tables["authorizer_users"].columns, "email")

	dropped, err := parseMigrations(append(slices.Clone(migrations), migration{
		version: len(migrations) + 1, name: "drift",
		statements: []string{
			`ALTER TABLE authorizer_users DROP COLUMN IF EXISTS email`,
			`DROP INDEX IF EXISTS idx_authorizer_users_email`,
		},
	}))
	require.NoError(t, err)
	assert.NotContains(t, dropped.tables["authorizer_users"].columns, "email")
	assert.NotContains(t, dropped.indexes, "idx_authorizer_users_email")

	_, err = parseMigrations([]migration{{version: 1, name: "x", statements: []string{`ALTER TABLE t RENAME TO u`}}})
	assert.Error(t, err, "an unknown statement shape must fail loudly, not be skipped")
}

func TestValidateMigrations(t *testing.T) {
	require.NoError(t, validateMigrations(migrations))
	assert.Error(t, validateMigrations([]migration{{version: 2, name: "gap", statements: []string{"x"}}}))
	assert.Error(t, validateMigrations([]migration{
		{version: 1, name: "a", statements: []string{"x"}},
		{version: 1, name: "dup", statements: []string{"x"}},
	}))
	assert.Error(t, validateMigrations([]migration{{version: 1, name: "empty"}}))
}
