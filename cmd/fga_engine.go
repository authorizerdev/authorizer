package cmd

import (
	"strings"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	fgaengine "github.com/authorizerdev/authorizer/internal/authorization/engine/openfga"
	"github.com/authorizerdev/authorizer/internal/config"
)

// initAuthzEngine initializes the embedded OpenFGA authorization engine from
// the --fga-store / --fga-store-url config, shared by the server (root) and
// the MCP subcommand. OpenFGA migrations run on boot for SQL stores
// (idempotent); the in-memory store needs none.
//
// Engine-init failure is deliberately NON-fatal: FGA is an optional
// subsystem, so a failure here (e.g. the DB user lacks DDL rights for the
// OpenFGA tables) logs loudly and returns a nil engine — fga_* and the
// permission APIs fail closed while core authentication keeps serving.
//
// The returned cleanup func is always non-nil and safe to defer; it closes
// the engine when one was created.
func initAuthzEngine(cfg *config.Config, log *zerolog.Logger) (engine.AuthorizationEngine, func()) {
	cleanup := func() {}
	fgaStore, fgaStoreURL, fgaEnabled := cfg.FGAStoreConfig()
	if !fgaEnabled {
		return nil, cleanup
	}
	runMigrations := !strings.EqualFold(fgaStore, fgaengine.StoreMemory)
	fgaEngine, err := fgaengine.New(
		&fgaengine.Config{
			Store:         fgaStore,
			StoreURL:      fgaStoreURL,
			StoreName:     cfg.OrganizationName,
			RunMigrations: runMigrations,
		},
		&fgaengine.Dependencies{Log: log},
	)
	if err != nil {
		log.Error().Err(err).
			Str("fga_store", fgaStore).
			Msg("failed to initialize OpenFGA authorization engine; fine-grained authorization is DISABLED (fail-closed) — core auth continues")
		return nil, cleanup
	}
	if closer, ok := fgaEngine.(interface{ Close() }); ok {
		cleanup = closer.Close
	}
	log.Info().
		Str("fga_store", fgaStore).
		Bool("reused_main_db", strings.TrimSpace(cfg.FGAStore) == "").
		Msg("OpenFGA authorization engine initialized (embedded)")
	return fgaEngine, cleanup
}
