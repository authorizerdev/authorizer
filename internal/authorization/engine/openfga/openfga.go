// Package openfga implements the engine.AuthorizationEngine SPI by embedding
// the OpenFGA server in-process (openfga v1.17.1). It supports an in-memory
// datastore (dev/tests) and persistent SQL datastores (SQLite single-node,
// Postgres/MySQL for HA) per the migration plan's deployment modes (§2.1).
//
// This package is additive (Phase 1). It does not replace the existing
// resource/scope/policy engine; both are selectable behind
// --authorization-engine. The principal-pinning, admin-gating, audit, and
// caching policy described in the plan live in the callers of this engine —
// this package is the thin, fail-closed adapter over OpenFGA.
package openfga

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/openfga/pkg/server"
	"github.com/openfga/openfga/pkg/storage"
	"github.com/openfga/openfga/pkg/storage/memory"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// Store kinds for the embedded datastore.
const (
	// StoreMemory selects the in-memory datastore (dev/tests only; non-durable).
	StoreMemory = "memory"
	// StoreSQLite selects the embedded SQLite datastore (single-node/dev).
	StoreSQLite = "sqlite"
	// StorePostgres selects an external Postgres datastore (HA).
	StorePostgres = "postgres"
	// StoreMySQL selects an external MySQL datastore (HA).
	StoreMySQL = "mysql"
)

// Config holds the parameters needed to construct the embedded OpenFGA engine.
//
// StoreID and ModelID are the OpenFGA-assigned ULIDs that Authorizer MUST
// persist itself (config or main DB) and pass back on every call — OpenFGA does
// not look stores/models up by name across restarts. When they are empty (e.g.
// memory store on first boot), the caller is expected to bootstrap via
// CreateStore + WriteModel and then persist the returned IDs.
type Config struct {
	// Store selects the datastore kind: memory|sqlite|postgres|mysql.
	Store string
	// StoreURL is the datastore connection URI (file: URI for sqlite, DSN for
	// postgres/mysql). Ignored for the memory store.
	StoreURL string
	// StoreName is the OpenFGA store name used when bootstrapping a new store.
	StoreName string
	// StoreID, when set, targets an existing OpenFGA store (skips CreateStore).
	StoreID string
	// ModelID, when set, targets an existing authorization model (skips the
	// need to write one before checks).
	ModelID string
	// RunMigrations, when true, runs the datastore migrations during Init for
	// SQL stores. For HA/serverless this MUST be false — migrations run as a
	// separate init job (§2.1) to avoid races and cold-start latency.
	RunMigrations bool
}

// Dependencies carries shared resources for constructing the engine.
type Dependencies struct {
	Log *zerolog.Logger
}

// engineImpl implements engine.AuthorizationEngine over an embedded OpenFGA
// server. storeID and modelID are mutated under mu when a store/model is
// bootstrapped at runtime (memory store / first boot).
type engineImpl struct {
	log     *zerolog.Logger
	srv     *server.Server
	ds      storage.OpenFGADatastore
	mu      sync.RWMutex
	storeID string
	modelID string
}

// Compile-time interface verification.
var _ engine.AuthorizationEngine = &engineImpl{}

// New constructs the embedded OpenFGA engine.
//
// Migrations are deliberately NOT run unconditionally: they run only when
// cfg.RunMigrations is true (single-node/dev). HA and serverless deployments
// must run migrate.RunMigrations as a separate init job and leave
// RunMigrations=false so engine boot assumes the schema already exists (§2.1).
//
// If cfg.StoreID is empty a new store is created and its ID exposed via
// StoreID(); callers should persist it. If cfg.ModelID is empty, callers must
// call WriteModel before issuing checks.
func New(cfg *Config, deps *Dependencies) (engine.AuthorizationEngine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("openfga.New: config is required")
	}
	if deps == nil || deps.Log == nil {
		return nil, fmt.Errorf("openfga.New: logger is required")
	}
	log := deps.Log.With().Str("component", "fga-engine").Logger()

	ds, err := newDatastore(cfg, &log)
	if err != nil {
		return nil, err
	}

	srv, err := server.NewServerWithOpts(server.WithDatastore(ds))
	if err != nil {
		ds.Close()
		return nil, fmt.Errorf("openfga.New: NewServerWithOpts: %w", err)
	}

	e := &engineImpl{
		log:     &log,
		srv:     srv,
		ds:      ds,
		storeID: cfg.StoreID,
		modelID: cfg.ModelID,
	}

	// Bootstrap a store if none was provided. The store ID is exposed via
	// StoreID() so the caller can persist it for subsequent boots.
	if e.storeID == "" {
		store, cErr := srv.CreateStore(context.Background(), &openfgav1.CreateStoreRequest{
			Name: storeNameOrDefault(cfg.StoreName),
		})
		if cErr != nil {
			srv.Close()
			ds.Close()
			return nil, fmt.Errorf("openfga.New: CreateStore: %w", cErr)
		}
		e.storeID = store.GetId()
		log.Info().Str("store_id", e.storeID).Msg("created new OpenFGA store; persist this ID")
	}

	return e, nil
}

// newDatastore opens the configured datastore.
//
// The memory store and all SQL stores (sqlite/postgres/mysql) are compiled into
// the default binary. OpenFGA's SQLite datastore uses modernc.org/sqlite, the
// same pure-Go driver that Authorizer's GORM SQLite dialect now targets (see
// internal/storage/db/sql/sqlitedialect). Because modernc.org/sqlite is the
// single registrant of the "sqlite" database/sql driver, embedding SQL-backed
// FGA alongside the GORM SQLite main DB no longer panics at startup with
// "sql: Register called twice for driver sqlite".
func newDatastore(cfg *Config, log *zerolog.Logger) (storage.OpenFGADatastore, error) {
	switch cfg.Store {
	case "", StoreMemory:
		return memory.New(), nil
	case StoreSQLite, StorePostgres, StoreMySQL:
		if cfg.StoreURL == "" {
			return nil, fmt.Errorf("openfga.New: --fga-store-url is required for store %q", cfg.Store)
		}
		return newSQLDatastore(cfg, log)
	default:
		return nil, fmt.Errorf("openfga.New: unsupported fga store %q (want memory|sqlite|postgres|mysql)", cfg.Store)
	}
}

func storeNameOrDefault(name string) string {
	if name == "" {
		return "authorizer"
	}
	return name
}

// StoreID returns the OpenFGA store ID this engine is bound to. Callers should
// persist it (config/main DB) so subsequent boots target the same store.
func (e *engineImpl) StoreID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.storeID
}

// ModelID returns the currently active authorization model ID, or empty if no
// model has been written yet. Callers should persist it alongside the store ID.
func (e *engineImpl) ModelID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.modelID
}

// Close releases the embedded server and datastore, flushing any WAL. It should
// be deferred at the construction site.
func (e *engineImpl) Close() {
	if e.srv != nil {
		e.srv.Close()
	}
	if e.ds != nil {
		e.ds.Close()
	}
}

// ids returns the current store and model IDs under the read lock.
func (e *engineImpl) ids() (storeID, modelID string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.storeID, e.modelID
}

// strconvItoa is a tiny helper to build correlation IDs for BatchCheck.
func strconvItoa(i int) string { return strconv.Itoa(i) }

// toProtoContextual converts engine contextual tuples to the OpenFGA wire type.
func toProtoContextual(ctxTuples []engine.ContextualTuple) *openfgav1.ContextualTupleKeys {
	if len(ctxTuples) == 0 {
		return nil
	}
	keys := make([]*openfgav1.TupleKey, 0, len(ctxTuples))
	for _, t := range ctxTuples {
		keys = append(keys, &openfgav1.TupleKey{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		})
	}
	return &openfgav1.ContextualTupleKeys{TupleKeys: keys}
}
