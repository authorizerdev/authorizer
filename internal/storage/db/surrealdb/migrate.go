package surrealdb

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// migrationsTable records which versions of the schema in schema.go a
// database has applied. The record id is the version.
const migrationsTable = "authorizer_schema_migrations"

// maxMigrationAttempts bounds how long a booting instance waits out another
// instance applying the same version.
const maxMigrationAttempts = 20

type migration struct {
	version    int
	name       string
	statements []string
}

// migrate applies every version in list the database has not recorded, in
// order. Each version's DDL and its record run in one transaction, so a failed
// version leaves neither behind and the next boot retries it.
//
// Concurrent boots are not serialised. Every statement is idempotent and the
// version record is the tie-breaker: an instance that loses the race fails on
// the existing version record (or a transaction conflict), finds the version
// recorded, and moves on.
func (p *provider) migrate(ctx context.Context, list []migration) error {
	if err := validateMigrations(list); err != nil {
		return err
	}
	if err := p.exec(ctx, "DEFINE TABLE IF NOT EXISTS "+migrationsTable+" SCHEMALESS;", nil); err != nil && !isConflict(err) {
		return fmt.Errorf("surrealdb: create %s: %w", migrationsTable, err)
	}
	for _, m := range list {
		applied, err := p.migrationApplied(ctx, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := p.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("surrealdb: migration %d (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}

func (p *provider) applyMigration(ctx context.Context, m migration) error {
	body := strings.Join(m.statements, ";\n") + ";\n" +
		"CREATE type::record($mt, $v) SET name = $n, applied_at = $t RETURN NONE;"
	var err error
	for attempt := 0; attempt < maxMigrationAttempts; attempt++ {
		err = p.transaction(ctx, body, map[string]any{
			"mt": migrationsTable, "v": m.version, "n": m.name, "t": time.Now().Unix(),
		})
		if err == nil {
			return nil
		}
		if !isRecordExists(err) && !isConflict(err) {
			return err // includes a unique-index violation: a real failure
		}
		// Lost the race to another instance? Its record settles it.
		if applied, aerr := p.migrationApplied(ctx, m.version); aerr == nil && applied {
			return nil
		}
		if isRecordExists(err) {
			return err // the record exists but is not this version's: not a race
		}
		base := time.Duration(attempt+1) * 50 * time.Millisecond
		time.Sleep(base + rand.N(base))
	}
	return err
}

func (p *provider) migrationApplied(ctx context.Context, version int) (bool, error) {
	res, err := run[bool](ctx, p, "RETURN record::exists(type::record($mt, $v));",
		map[string]any{"mt": migrationsTable, "v": version})
	if err != nil {
		return false, fmt.Errorf("surrealdb: read %s: %w", migrationsTable, err)
	}
	return res != nil && len(*res) > 0 && (*res)[0].Result, nil
}

// validateMigrations rejects a list whose versions are not 1, 2, 3, ... in
// order, or that has an empty version.
func validateMigrations(list []migration) error {
	for i, m := range list {
		if m.version != i+1 {
			return fmt.Errorf("surrealdb: migration at index %d has version %d, want %d", i, m.version, i+1)
		}
		if len(m.statements) == 0 {
			return fmt.Errorf("surrealdb: migration %d (%s) has no statements", m.version, m.name)
		}
	}
	return nil
}
