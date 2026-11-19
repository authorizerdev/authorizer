package serenedb

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// migrationsTable records which versions of the DDL in schema.go a database
// has applied.
const migrationsTable = "authorizer_schema_migrations"

// maxMigrationAttempts bounds how long a booting instance waits out another
// instance applying the same version (see applyMigration).
const maxMigrationAttempts = 20

// migration is one versioned step of the SereneDB schema. See schema.go for
// the rules that keep it safe to change.
type migration struct {
	version    int
	name       string
	statements []string
}

// migrate brings db up to the schema in schema.go. It is the sql.Options
// Migrate hook, replacing GORM AutoMigrate for SereneDB.
func migrate(db *gorm.DB) error {
	return runMigrations(db, migrations)
}

// runMigrations applies every version in list that db has not recorded yet,
// in order, each in its own transaction together with the row that records
// it. A failed version therefore leaves neither its DDL nor its record behind
// (SereneDB DDL is transactional), and the next boot retries it.
//
// SereneDB has no advisory locks, so concurrent boots are not serialised.
// Instead every statement is idempotent and the version row is the
// tie-breaker; see applyMigration.
func runMigrations(db *gorm.DB, list []migration) error {
	if err := validateMigrations(list); err != nil {
		return err
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + migrationsTable + ` (
		version bigint,
		name text,
		applied_at bigint,
		PRIMARY KEY (version)
	)`).Error; err != nil {
		return fmt.Errorf("serenedb: create %s: %w", migrationsTable, err)
	}

	applied, err := appliedVersions(db)
	if err != nil {
		return err
	}
	for _, m := range list {
		if applied[m.version] {
			continue
		}
		if err := applyMigration(db, m); err != nil {
			return fmt.Errorf("serenedb: migration %d (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}

// applyMigration runs one version and records it, in a single transaction.
//
// Another instance booting at the same moment makes this fail with a write
// conflict (40001: both are creating the same catalog objects) or a duplicate
// version row (23505). Neither is a real failure. Once the other instance
// commits, the version is recorded and there is nothing left to do; until it
// does, retrying is safe because every statement is idempotent.
func applyMigration(db *gorm.DB, m migration) error {
	var err error
	for attempt := 0; attempt < maxMigrationAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 50 * time.Millisecond)
			applied, rerr := appliedVersions(db)
			if rerr != nil {
				return rerr
			}
			if applied[m.version] {
				return nil
			}
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			for i, stmt := range m.statements {
				if err := tx.Exec(stmt).Error; err != nil {
					return fmt.Errorf("statement %d: %w", i+1, err)
				}
			}
			return tx.Exec(`INSERT INTO `+migrationsTable+` (version, name, applied_at) VALUES (?, ?, ?)`,
				m.version, m.name, time.Now().Unix()).Error
		})
		if err == nil || (!uniqueViolation(err) && !sqlState(err, "40001")) {
			return err
		}
	}
	return err
}

func appliedVersions(db *gorm.DB) (map[int]bool, error) {
	var versions []int
	if err := db.Raw(`SELECT version FROM ` + migrationsTable).Scan(&versions).Error; err != nil {
		return nil, fmt.Errorf("serenedb: read %s: %w", migrationsTable, err)
	}
	applied := make(map[int]bool, len(versions))
	for _, v := range versions {
		applied[v] = true
	}
	return applied, nil
}

// validateMigrations rejects a list whose versions are not strictly
// increasing from 1. Applying out of order, or two steps sharing a version,
// would make the recorded state ambiguous.
func validateMigrations(list []migration) error {
	for i, m := range list {
		if m.version != i+1 {
			return fmt.Errorf("serenedb: migration at index %d has version %d, want %d", i, m.version, i+1)
		}
		if len(m.statements) == 0 {
			return fmt.Errorf("serenedb: migration %d (%s) has no statements", m.version, m.name)
		}
	}
	return nil
}
