// Package serenedb adapts the SQL storage provider to SereneDB, a search-OLAP
// engine that speaks the PostgreSQL wire protocol. It reuses the SQL provider
// wholesale and overrides only the two places where SereneDB's SQL differs from
// PostgreSQL's:
//
//  1. Migration. GORM AutoMigrate reconciles column types on every boot, but
//     SereneDB reports varchar(n) as text and refuses ALTER COLUMN ... TYPE on
//     an indexed column, so the second startup fails. migrate below creates what
//     is missing and never alters what exists.
//  2. Upserts with an explicit conflict target (see upsert.go).
//
// ponytail: SereneDB's MVCC is optimistic, so contended same-row writes abort
// with SQLSTATE 40001 instead of blocking. Only the two upserts here retry it;
// every other write path inherits the SQL provider unchanged and will surface
// 40001 to the caller under the same contention. Move the retry to a GORM
// callback if that shows up in practice.
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
	base, err := sql.NewProviderWithMigrate(cfg, &sql.Dependencies{Log: deps.Log}, migrate)
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
		for name := range stmt.Schema.ParseIndexes() {
			if !m.HasIndex(model, name) {
				if err := m.CreateIndex(model, name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
