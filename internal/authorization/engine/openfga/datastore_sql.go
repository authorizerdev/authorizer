package openfga

import (
	"fmt"
	"time"

	"github.com/openfga/openfga/pkg/storage"
	"github.com/openfga/openfga/pkg/storage/migrate"
	"github.com/openfga/openfga/pkg/storage/mysql"
	"github.com/openfga/openfga/pkg/storage/postgres"
	"github.com/openfga/openfga/pkg/storage/sqlcommon"
	"github.com/openfga/openfga/pkg/storage/sqlite"
	"github.com/rs/zerolog"
)

// Default timeouts for the migration bootstrap step.
const (
	defaultMigrateTimeout     = 30 * time.Second
	defaultMigratePingTimeout = 5 * time.Second
)

// newSQLDatastore opens a SQL-backed OpenFGA datastore against an (optionally
// just-migrated) schema. The SQLite datastore uses modernc.org/sqlite (pure-Go),
// which is the single registrant of the "sqlite" database/sql driver shared with
// Authorizer's GORM SQLite path (see internal/storage/db/sql/sqlitedialect).
func newSQLDatastore(cfg *Config, log *zerolog.Logger) (storage.OpenFGADatastore, error) {
	if cfg.RunMigrations {
		if err := runMigrations(cfg, log); err != nil {
			return nil, err
		}
	}
	sc := sqlcommon.NewConfig()
	switch cfg.Store {
	case StoreSQLite:
		ds, err := sqlite.New(cfg.StoreURL, sc)
		if err != nil {
			return nil, fmt.Errorf("openfga.New: sqlite.New: %w", err)
		}
		return ds, nil
	case StorePostgres:
		ds, err := postgres.New(cfg.StoreURL, sc)
		if err != nil {
			return nil, fmt.Errorf("openfga.New: postgres.New: %w", err)
		}
		return ds, nil
	case StoreMySQL:
		ds, err := mysql.New(cfg.StoreURL, sc)
		if err != nil {
			return nil, fmt.Errorf("openfga.New: mysql.New: %w", err)
		}
		return ds, nil
	default:
		return nil, fmt.Errorf("openfga.New: unsupported sql store %q", cfg.Store)
	}
}

// runMigrations runs the OpenFGA datastore migrations (idempotent, embedded).
// For HA/serverless this must NOT be called on boot — run it as a separate init
// job (§2.1).
func runMigrations(cfg *Config, log *zerolog.Logger) error {
	log.Info().Str("engine", cfg.Store).Msg("running OpenFGA datastore migrations")
	if err := migrate.RunMigrations(migrate.MigrationConfig{
		Engine:      cfg.Store,
		URI:         cfg.StoreURL,
		Timeout:     defaultMigrateTimeout,
		PingTimeout: defaultMigratePingTimeout,
	}); err != nil {
		return fmt.Errorf("openfga.New: RunMigrations(%s): %w", cfg.Store, err)
	}
	return nil
}
