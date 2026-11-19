// Package serenedb adapts the SQL storage provider to SereneDB, a search-OLAP
// engine that speaks the PostgreSQL wire protocol. It reuses the SQL provider
// wholesale and changes only two things:
//
//  1. Migration. GORM AutoMigrate reconciles column types on every boot, but
//     SereneDB reports varchar(n) as text and refuses ALTER COLUMN ... TYPE on
//     an indexed column, so the second startup fails. migrate below creates what
//     is missing and never alters what exists. Still required as of 26.11.2.
//  2. Write conflicts. SereneDB's MVCC is optimistic, so contended same-row
//     writes abort with SQLSTATE 40001 instead of blocking. That is retried at
//     the connection pool (see connpool.go) so every write path gets it — but
//     only for autocommit statements. The SQL provider's four explicit
//     Transaction() call sites still surface 40001 under sustained same-row
//     contention. Still required as of 26.11.2.
//
// Releases before 26.11 also resolved ON CONFLICT (cols) only against an inline
// UNIQUE constraint, never a CREATE UNIQUE INDEX, which needed overrides for the
// two upserting methods. 26.11 fixed that, so the SQL provider's upserts run
// unchanged; TestAddVerificationRequestUpsert and
// TestAddAuthenticatorConcurrentEnrollment pin it.
package serenedb

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"

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
	})
	if err != nil {
		return nil, err
	}
	return &provider{Provider: base}, nil
}

// migrate creates missing tables, columns and indexes, and nothing else.
// SereneDB supports CREATE TABLE, ALTER TABLE ADD COLUMN and CREATE INDEX, but
// not the ALTER COLUMN ... TYPE / SET NOT NULL / SET DEFAULT that GORM's
// AutoMigrate issues when reconciling an existing table.
func migrate(db *gorm.DB) error {
	m := db.Migrator()
	for _, model := range sql.Models() {
		if !m.HasTable(model) {
			if err := m.CreateTable(model); err != nil {
				return err
			}
			continue
		}
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return err
		}
		for _, field := range stmt.Schema.Fields {
			if field.DBName == "" || field.IgnoreMigration {
				continue
			}
			if !m.HasColumn(model, field.DBName) {
				if err := m.AddColumn(model, field.DBName); err != nil {
					return err
				}
			}
		}
		for _, idx := range stmt.Schema.ParseIndexes() {
			if !m.HasIndex(model, idx.Name) {
				if err := m.CreateIndex(model, idx.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
