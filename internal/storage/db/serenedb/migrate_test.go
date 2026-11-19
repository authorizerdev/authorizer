package serenedb

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/db/sql"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// freshDatabase creates an empty database for one test and returns its URL.
// Migration tests need a clean slate: the shared test database has long since
// been migrated by every other test.
func freshDatabase(t *testing.T) string {
	t.Helper()
	requireSereneDB(t)
	admin := openRaw(t, testDBURL)
	name := "authorizer_mig_" + strings.ReplaceAll(uuid.NewString()[:13], "-", "")
	require.NoError(t, admin.Exec("CREATE DATABASE "+name).Error)
	t.Cleanup(func() {
		if err := admin.Exec("DROP DATABASE " + name).Error; err != nil {
			t.Logf("drop %s: %v", name, err)
		}
	})
	return strings.TrimSuffix(testDBURL, "/postgres") + "/" + name
}

// openRaw opens a GORM handle configured the way the provider configures its
// own, without running any migration.
func openRaw(t *testing.T, url string) *gorm.DB {
	t.Helper()
	raw, err := stdsql.Open("pgx", url)
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: newRetryPool(raw)}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Discard,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	return db
}

func recordedVersions(t *testing.T, db *gorm.DB) []int {
	t.Helper()
	var v []int
	require.NoError(t, db.Raw("SELECT version FROM "+migrationsTable+" ORDER BY version").Scan(&v).Error)
	return v
}

func allVersions() []int {
	v := make([]int, len(migrations))
	for i, m := range migrations {
		v[i] = m.version
	}
	return v
}

// TestMigrateFreshDatabase applies the DDL to an empty database and checks the
// result against the models through the live catalog — the counterpart of
// TestSchemaDDLMatchesModels, proving SereneDB built what schema.go declares.
func TestMigrateFreshDatabase(t *testing.T) {
	url := freshDatabase(t)
	p := newProviderAt(t, url)
	defer p.Close() //nolint:errcheck
	db := p.DB()

	assert.Equal(t, allVersions(), recordedVersions(t, db))

	m := db.Migrator()
	for _, model := range sql.Models() {
		ms := parseModel(t, model)
		require.True(t, m.HasTable(model), "table %s", ms.Table)
		for _, f := range ms.Fields {
			if f.DBName != "" && !f.IgnoreMigration {
				assert.True(t, m.HasColumn(model, f.DBName), "column %s.%s", ms.Table, f.DBName)
			}
		}
		for _, idx := range ms.ParseIndexes() {
			assert.True(t, m.HasIndex(model, idx.Name), "index %s on %s", idx.Name, ms.Table)
		}
	}

	// Column DEFAULTs are load-bearing: GORM leaves a false IsActive out of the
	// INSERT, so the row is whatever the DDL default says.
	ctx := context.Background()
	user, err := p.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef(uuid.NewString() + "@authorizer.dev"),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)
	got, err := p.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, got.IsActive, "is_active must default to true")
}

// TestMigrateRestartIsNoop is the path that broke under GORM AutoMigrate: the
// second boot against an existing schema. It must succeed, change nothing and
// keep the data.
func TestMigrateRestartIsNoop(t *testing.T) {
	url := freshDatabase(t)
	p := newProviderAt(t, url)
	ctx := context.Background()
	user, err := p.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef(uuid.NewString() + "@authorizer.dev"),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)
	require.NoError(t, p.Close())

	for i := 0; i < 3; i++ {
		p := newProviderAt(t, url)
		_, err := p.GetUserByID(ctx, user.ID)
		require.NoError(t, err, "boot %d lost data", i+2)
		assert.Equal(t, allVersions(), recordedVersions(t, p.DB()), "boot %d re-recorded a version", i+2)
		require.NoError(t, p.Close())
	}
}

// TestMigrateAppliesNewVersion is the upgrade path: a database on the current
// schema gets a later migration on the next boot, keeps its rows, and records
// the new version once.
func TestMigrateAppliesNewVersion(t *testing.T) {
	db := openRaw(t, freshDatabase(t))
	require.NoError(t, runMigrations(db, migrations))
	require.NoError(t, db.Exec(`INSERT INTO authorizer_users (id, email) VALUES ('u1', 'a@b.c')`).Error)

	next := append(slices.Clone(migrations), migration{
		version: len(migrations) + 1,
		name:    "test: add column and index",
		statements: []string{
			`ALTER TABLE authorizer_users ADD COLUMN IF NOT EXISTS zz_probe text DEFAULT 'x'`,
			`CREATE INDEX IF NOT EXISTS idx_zz_probe ON authorizer_users (zz_probe)`,
		},
	})
	require.NoError(t, runMigrations(db, next))
	require.NoError(t, runMigrations(db, next), "re-running an applied list must be a no-op")

	assert.Equal(t, append(allVersions(), len(next)), recordedVersions(t, db))
	var email string
	require.NoError(t, db.Raw(`SELECT email FROM authorizer_users WHERE id = 'u1'`).Scan(&email).Error)
	assert.Equal(t, "a@b.c", email)
	assert.True(t, db.Migrator().HasColumn("authorizer_users", "zz_probe"))
}

// TestMigrateFailedVersionRollsBack: a migration that fails part-way must
// leave neither its earlier statements nor its version row behind, so the
// next boot (with the migration fixed) starts it from scratch.
func TestMigrateFailedVersionRollsBack(t *testing.T) {
	db := openRaw(t, freshDatabase(t))
	bad := append(slices.Clone(migrations), migration{
		version: len(migrations) + 1,
		name:    "test: fails half-way",
		statements: []string{
			`CREATE TABLE IF NOT EXISTS zz_half (id text, PRIMARY KEY (id))`,
			`THIS IS NOT SQL`,
		},
	})
	err := runMigrations(db, bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("migration %d", len(bad)))
	assert.Contains(t, err.Error(), "statement 2")

	assert.Equal(t, allVersions(), recordedVersions(t, db), "earlier versions stay applied, the failed one is not recorded")
	assert.False(t, db.Migrator().HasTable("zz_half"), "the failed version's first statement must roll back")

	fixed := slices.Clone(bad)
	fixed[len(fixed)-1].statements = []string{`CREATE TABLE IF NOT EXISTS zz_half (id text, PRIMARY KEY (id))`}
	require.NoError(t, runMigrations(db, fixed))
	assert.True(t, db.Migrator().HasTable("zz_half"))
	assert.Equal(t, append(allVersions(), len(fixed)), recordedVersions(t, db))
}

// TestMigrateConcurrentBoot: several instances starting against one empty
// database at once. SereneDB has no advisory locks, so they race; every one
// must come up, and each version must be recorded exactly once.
func TestMigrateConcurrentBoot(t *testing.T) {
	url := freshDatabase(t)
	const instances = 4
	errs := make([]error, instances)
	var wg sync.WaitGroup
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw, err := stdsql.Open("pgx", url)
			if err != nil {
				errs[i] = err
				return
			}
			defer raw.Close() //nolint:errcheck
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: newRetryPool(raw)}), &gorm.Config{
				SkipDefaultTransaction: true, Logger: logger.Discard,
			})
			if err != nil {
				errs[i] = err
				return
			}
			errs[i] = runMigrations(db, migrations)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "instance %d", i)
	}
	assert.Equal(t, allVersions(), recordedVersions(t, openRaw(t, url)))
}
