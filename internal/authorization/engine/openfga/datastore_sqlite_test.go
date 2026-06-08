package openfga

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	sqlitedialect "github.com/authorizerdev/authorizer/internal/storage/db/sql/sqlitedialect"
)

// TestOpenFGAEngine_SQLiteStore_EndToEnd proves that the embedded SQL-backed
// OpenFGA engine runs in-process against a real on-disk SQLite database — built
// into the DEFAULT binary (no fga_sql build tag) — alongside Authorizer's GORM
// SQLite path, without the historical "sql: Register called twice for driver
// sqlite" panic.
//
// Both code paths use modernc.org/sqlite as the single registrant of the
// "sqlite" database/sql driver: OpenFGA's sqlite datastore opens it directly,
// and Authorizer's GORM dialect (internal/storage/db/sql/sqlitedialect) targets
// the same driver. This test exercises both in one process.
func TestOpenFGAEngine_SQLiteStore_EndToEnd(t *testing.T) {
	ctx := context.Background()
	log := zerolog.New(os.Stderr)

	dir := t.TempDir()

	// 1) Open a GORM SQLite DB via Authorizer's local dialect to assert the two
	//    SQLite consumers coexist in one process (no double-registration panic).
	gormDBPath := filepath.Join(dir, "authorizer-main.db")
	gormDB, err := gorm.Open(
		sqlitedialect.Open(gormDBPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"),
		&gorm.Config{},
	)
	require.NoError(t, err, "GORM SQLite (modernc) must open without driver conflict")
	require.NoError(t, gormDB.Exec("CREATE TABLE IF NOT EXISTS probe (id INTEGER PRIMARY KEY)").Error)
	require.NoError(t, gormDB.Exec("INSERT INTO probe (id) VALUES (1)").Error)
	var probeCount int64
	require.NoError(t, gormDB.Raw("SELECT COUNT(*) FROM probe").Scan(&probeCount).Error)
	assert.Equal(t, int64(1), probeCount, "GORM SQLite path works")

	// 2) Construct the embedded OpenFGA engine against a SQLite file store, with
	//    migrations run on boot (single-node mode).
	fgaDBPath := filepath.Join(dir, "openfga.db")
	fgaURI := fmt.Sprintf("file:%s", fgaDBPath)

	eng, err := New(&Config{
		Store:         StoreSQLite,
		StoreURL:      fgaURI,
		RunMigrations: true,
	}, &Dependencies{Log: &log})
	require.NoError(t, err, "embedded OpenFGA SQLite engine must construct (migrations + open)")
	require.NotNil(t, eng)

	impl, ok := eng.(*engineImpl)
	require.True(t, ok)
	t.Cleanup(impl.Close)

	assert.NotEmpty(t, impl.StoreID(), "store ID bootstrapped on the SQLite store")

	// 3) Write a model + tuples and Check — proving the persistent store works.
	modelID, err := eng.WriteModel(ctx, testModel)
	require.NoError(t, err)
	assert.NotEmpty(t, modelID)

	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
		{User: "user:erin", Relation: "viewer", Object: "document:1"},
		{User: "user:erin", Relation: "blocked", Object: "document:1"},
	}))

	allowed, err := eng.Check(ctx, "user:alice", "can_view", "document:1")
	require.NoError(t, err)
	assert.True(t, allowed, "alice is a viewer and not blocked")

	allowed, err = eng.Check(ctx, "user:erin", "can_view", "document:1")
	require.NoError(t, err)
	assert.False(t, allowed, "erin is blocked despite being a viewer")

	allowed, err = eng.Check(ctx, "user:bob", "can_view", "document:1")
	require.NoError(t, err)
	assert.False(t, allowed, "bob has no grant")

	// 4) Confirm data was actually persisted to the SQLite file on disk.
	info, statErr := os.Stat(fgaDBPath)
	require.NoError(t, statErr, "OpenFGA SQLite db file must exist on disk")
	assert.Positive(t, info.Size(), "OpenFGA SQLite db file must be non-empty")
}
