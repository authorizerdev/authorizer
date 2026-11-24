package surrealdb

import (
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

// These tests need no database: they run in `make test`.

type ddlIndex struct {
	table  string
	cols   []string
	unique bool
}

var (
	reDefineTable = regexp.MustCompile(`^DEFINE TABLE IF NOT EXISTS (\w+) SCHEMALESS$`)
	reDefineIndex = regexp.MustCompile(`^DEFINE INDEX IF NOT EXISTS (\w+) ON (\w+) FIELDS ([\w, ]+?)( UNIQUE)?$`)
)

// parseSchema replays the migrations' DDL. It understands exactly the
// statement shapes schema.go uses and fails on anything else, so a new shape
// has to be taught here before it can ship.
func parseSchema(t *testing.T, list []migration) (map[string]bool, map[string]ddlIndex) {
	t.Helper()
	tables := map[string]bool{}
	indexes := map[string]ddlIndex{}
	for _, m := range list {
		for _, stmt := range m.statements {
			stmt = strings.Join(strings.Fields(stmt), " ")
			switch {
			case reDefineTable.MatchString(stmt):
				tables[reDefineTable.FindStringSubmatch(stmt)[1]] = true
			case reDefineIndex.MatchString(stmt):
				g := reDefineIndex.FindStringSubmatch(stmt)
				require.NotContains(t, indexes, g[1], "index %s defined twice", g[1])
				var cols []string
				for _, c := range strings.Split(g[3], ",") {
					cols = append(cols, strings.TrimSpace(c))
				}
				indexes[g[1]] = ddlIndex{table: g[2], cols: cols, unique: g[4] != ""}
			default:
				t.Fatalf("migration %d: statement shape not understood by the parity test (extend parseSchema): %s", m.version, stmt)
			}
		}
	}
	return tables, indexes
}

// TestSchemaMatchesSQLIndexes keeps SurrealDB's indexes identical to the SQL
// models' GORM index tags — same names, columns, order and uniqueness — so
// lookups and uniqueness behave the same on every backend. Adding an index or
// a model on the SQL side fails this until schema.go has a migration for it.
func TestSchemaMatchesSQLIndexes(t *testing.T) {
	tables, got := parseSchema(t, migrations)

	want := map[string]ddlIndex{}
	wantTables := map[string]bool{}
	for _, model := range sql.Models() {
		s, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{TablePrefix: schemas.Prefix})
		require.NoError(t, err)
		wantTables[s.Table] = true
		for _, idx := range s.ParseIndexes() {
			var cols []string
			for _, f := range idx.Fields {
				cols = append(cols, f.DBName)
			}
			want[idx.Name] = ddlIndex{table: s.Table, cols: cols, unique: idx.Class == "UNIQUE"}
		}
		for _, f := range s.Fields {
			if f.Unique && !f.PrimaryKey {
				want[schema.NamingStrategy{}.UniqueName(s.Table, f.DBName)] = ddlIndex{table: s.Table, cols: []string{f.DBName}, unique: true}
			}
		}
	}

	assert.Equal(t, wantTables, tables, "schema.go must define exactly the SQL models' tables")
	for name, w := range want {
		g, ok := got[name]
		if assert.True(t, ok, "index %s on %s is missing from schema.go", name, w.table) {
			assert.Equal(t, w, g, "index %s", name)
		}
	}
	for name := range got {
		assert.Contains(t, want, name, "schema.go defines index %s, which no SQL model declares", name)
	}
}

func TestValidateMigrations(t *testing.T) {
	require.NoError(t, validateMigrations(migrations))
	assert.Error(t, validateMigrations([]migration{{version: 2, name: "gap", statements: []string{"x"}}}))
	assert.Error(t, validateMigrations([]migration{{version: 1, name: "a", statements: []string{"x"}}, {version: 1, name: "dup", statements: []string{"x"}}}))
	assert.Error(t, validateMigrations([]migration{{version: 1, name: "empty"}}))
	assert.True(t, slices.IsSortedFunc(migrations, func(a, b migration) int { return a.version - b.version }))
}
