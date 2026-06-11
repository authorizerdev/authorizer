// Package openfga implements the engine.AuthorizationEngine SPI by embedding
// the OpenFGA server in-process (openfga v1.17.1). It supports an in-memory
// datastore (dev/tests) and persistent SQL datastores (SQLite single-node,
// Postgres/MySQL for HA) per the migration plan's deployment modes (§2.1).
//
// The principal-pinning, admin-gating, audit, and caching policy described in
// the plan live in the callers of this engine — this package is the thin,
// fail-closed adapter over OpenFGA.
package openfga

import (
	"context"
	"fmt"
	"sync"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/openfga/pkg/server"
	"github.com/openfga/openfga/pkg/storage"
	"github.com/openfga/openfga/pkg/storage/memory"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/wrapperspb"

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
// StoreID and ModelID are the OpenFGA-assigned ULIDs. They normally stay
// empty: on boot the engine recovers the existing store by name (StoreName)
// and adopts the latest written authorization model, so persistent datastores
// survive restarts without the caller persisting any IDs. Set them only to
// pin a specific store/model explicitly. Note: the store is found by exact
// name, so changing StoreName (organization name) starts a fresh store.
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
	log       *zerolog.Logger
	srv       *server.Server
	ds        storage.OpenFGADatastore
	mu        sync.RWMutex
	storeID   string
	modelID   string
	storeName string
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
// If cfg.StoreID is empty the engine binds to the existing store matching
// cfg.StoreName (restart continuity) or creates one when none exists. If
// cfg.ModelID is empty the latest written model in the store is adopted;
// when the store has no model yet, callers must WriteModel before checks.
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
		log:       &log,
		srv:       srv,
		ds:        ds,
		storeID:   cfg.StoreID,
		modelID:   cfg.ModelID,
		storeName: cfg.StoreName,
	}

	// Bind to a store. An explicit cfg.StoreID wins; otherwise reuse the
	// existing store with this name — OpenFGA assigns store ULIDs and has no
	// name lookup of its own, so without this scan every boot would create a
	// fresh store and orphan the previous model and tuples on persistent
	// datastores. Only when no store with the name exists is one created.
	if e.storeID == "" {
		name := storeNameOrDefault(cfg.StoreName)
		existing, fErr := findStoreByName(srv, name)
		if fErr != nil {
			srv.Close()
			ds.Close()
			return nil, fmt.Errorf("openfga.New: ListStores: %w", fErr)
		}
		if existing != "" {
			e.storeID = existing
			log.Info().Str("store_id", e.storeID).Str("store_name", name).Msg("reusing existing OpenFGA store")
		} else {
			store, cErr := srv.CreateStore(context.Background(), &openfgav1.CreateStoreRequest{Name: name})
			if cErr != nil {
				srv.Close()
				ds.Close()
				return nil, fmt.Errorf("openfga.New: CreateStore: %w", cErr)
			}
			e.storeID = store.GetId()
			log.Info().Str("store_id", e.storeID).Str("store_name", name).Msg("created new OpenFGA store")
		}
	}

	// Adopt the latest written model so checks keep working after a restart
	// without the caller persisting the model ID.
	if e.modelID == "" {
		mID, mErr := latestModelID(srv, e.storeID)
		if mErr != nil {
			srv.Close()
			ds.Close()
			return nil, fmt.Errorf("openfga.New: ReadAuthorizationModels: %w", mErr)
		}
		if mID != "" {
			e.modelID = mID
			log.Info().Str("model_id", mID).Msg("adopted latest authorization model from store")
		}
	}

	return e, nil
}

// findStoreByName pages through ListStores and returns the ID of the store
// with the exact given name, or empty when none exists. OpenFGA has no
// lookup-by-name API, so restart continuity depends on this scan.
func findStoreByName(srv *server.Server, name string) (string, error) {
	token := ""
	for {
		res, err := srv.ListStores(context.Background(), &openfgav1.ListStoresRequest{
			PageSize:          wrapperspb.Int32(100),
			ContinuationToken: token,
		})
		if err != nil {
			return "", err
		}
		for _, s := range res.GetStores() {
			if s.GetName() == name {
				return s.GetId(), nil
			}
		}
		token = res.GetContinuationToken()
		if token == "" {
			return "", nil
		}
	}
}

// latestModelID returns the most recent authorization model ID in the store,
// or empty when no model has been written yet. ReadAuthorizationModels
// returns models newest-first.
func latestModelID(srv *server.Server, storeID string) (string, error) {
	res, err := srv.ReadAuthorizationModels(context.Background(), &openfgav1.ReadAuthorizationModelsRequest{
		StoreId:  storeID,
		PageSize: wrapperspb.Int32(1),
	})
	if err != nil {
		return "", err
	}
	models := res.GetAuthorizationModels()
	if len(models) == 0 {
		return "", nil
	}
	return models[0].GetId(), nil
}

// Reset deletes the current store (model + all versions + tuples) and creates a
// fresh empty store. The new store has no model — callers must WriteModel again
// before checks. Destructive; admin-gated by callers.
func (e *engineImpl) Reset(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.storeID != "" {
		if _, err := e.srv.DeleteStore(ctx, &openfgav1.DeleteStoreRequest{StoreId: e.storeID}); err != nil {
			return fmt.Errorf("openfga.Reset: DeleteStore: %w", err)
		}
	}
	store, err := e.srv.CreateStore(ctx, &openfgav1.CreateStoreRequest{
		Name: storeNameOrDefault(e.storeName),
	})
	if err != nil {
		return fmt.Errorf("openfga.Reset: CreateStore: %w", err)
	}
	e.storeID = store.GetId()
	e.modelID = ""
	e.log.Info().Str("store_id", e.storeID).Msg("FGA store reset; new empty store created")
	return nil
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

// StoreID returns the OpenFGA store ID this engine is bound to. Subsequent
// boots recover the same store by name, so persisting this is optional.
func (e *engineImpl) StoreID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.storeID
}

// ModelID returns the currently active authorization model ID, or empty if no
// model has been written yet. Boots adopt the store's latest model automatically.
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
