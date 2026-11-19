// Package serenedb adapts the SQL storage provider to SereneDB, a search-OLAP
// engine that speaks the PostgreSQL wire protocol. Queries run through the SQL
// provider unchanged; two things differ:
//
//  1. Schema. GORM AutoMigrate is replaced by hand-written, versioned DDL
//     (schema.go, applied by migrate.go). AutoMigrate reconciles column types
//     on every boot, and SereneDB reports varchar(n) as text, so it issued an
//     ALTER COLUMN ... TYPE that SereneDB refuses on an indexed column and the
//     second startup failed. The DDL declares the types SereneDB really has,
//     so there is nothing to reconcile.
//  2. Write conflicts. SereneDB's MVCC is optimistic: contended writes to one
//     row abort with SQLSTATE 40001 instead of blocking. Autocommit
//     statements are retried at the connection pool (connpool.go) and explicit
//     transactions are replayed whole by the SQL provider (RetryTransactions).
//
// Releases before 26.11 also resolved ON CONFLICT (cols) only against an inline
// UNIQUE constraint, never a CREATE UNIQUE INDEX. 26.11 fixed that, and the DDL
// declares unique keys as inline constraints regardless.
package serenedb

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage/db/sql"
)

// Dependencies struct for the serenedb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	*sql.Provider
}

// NewProvider returns a new SereneDB provider
func NewProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	base, err := sql.NewProviderWithOptions(cfg, &sql.Dependencies{Log: deps.Log}, sql.Options{
		Migrate:                migrate,
		WrapPool:               newRetryPool,
		SkipDefaultTransaction: true,
		RetryTransactions:      true,
	})
	if err != nil {
		return nil, err
	}
	return &provider{Provider: base}, nil
}
