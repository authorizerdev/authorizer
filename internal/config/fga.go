package config

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// FGAStoreConfig resolves the OpenFGA datastore used by fine-grained
// authorization (FGA). Authorizer embeds OpenFGA in-process and, in the common
// case, reuses the main database — when it is one OpenFGA supports (sqlite,
// postgres, mysql/mariadb) — so no extra flags are needed.
//
// When the main database is NOT OpenFGA-compatible (e.g. mongodb, dynamodb,
// cassandra, couchbase, arangodb, sqlserver), the operator must set FGAStore
// (and FGAStoreURL) explicitly to point at a SQL store; otherwise FGA is
// disabled (enabled=false) and the engine is not constructed.
//
// An explicit FGAStore always overrides the derived value — e.g. to use a
// dedicated FGA database even when the main DB is SQL, or 'memory' for tests.
func (c *Config) FGAStoreConfig() (store string, url string, enabled bool) {
	// Explicit override wins over the main-DB-derived store.
	if s := strings.ToLower(strings.TrimSpace(c.FGAStore)); s != "" {
		if s == "sqlite" {
			return "sqlite", toSQLiteFileURI(c.FGAStoreURL), true
		}
		return s, c.FGAStoreURL, true
	}

	// Derive from the main database when OpenFGA supports it. Postgres- and
	// mysql-compatible variants beyond these (cockroachdb, yugabyte, libsql,
	// planetscale) are intentionally NOT auto-mapped — they require an explicit
	// --fga-store to avoid silent incompatibilities.
	switch c.DatabaseType {
	case constants.DbTypePostgres:
		return "postgres", c.DatabaseURL, true
	case constants.DbTypeMysql, constants.DbTypeMariaDB:
		return "mysql", c.DatabaseURL, true
	case constants.DbTypeSqlite:
		return "sqlite", toSQLiteFileURI(c.DatabaseURL), true
	default:
		return "", "", false
	}
}

// toSQLiteFileURI converts a SQLite path/URL into the file: URI OpenFGA's
// datastore expects. Authorizer stores a bare path (e.g. "data.db"); OpenFGA
// wants "file:data.db". Already-prefixed values are returned unchanged.
func toSQLiteFileURI(dbURL string) string {
	dbURL = strings.TrimSpace(dbURL)
	if dbURL == "" || strings.HasPrefix(dbURL, "file:") {
		return dbURL
	}
	return "file:" + dbURL
}
