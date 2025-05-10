package in_memory

import (
	"sync"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store/in_memory/stores"
)

// Dependencies struct for in_memory store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config          *config.Config
	dependencies    *Dependencies
	mutex           sync.Mutex
	sessionStore    *stores.SessionStore
	mfasessionStore *stores.SessionStore
	stateStore      *stores.StateStore
}

// NewInMemoryStore returns a new in-memory store.
func NewInMemoryProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	return &provider{
		config:          cfg,
		dependencies:    deps,
		mutex:           sync.Mutex{},
		sessionStore:    stores.NewSessionStore(),
		mfasessionStore: stores.NewSessionStore(),
		stateStore:      stores.NewStateStore(),
	}, nil
}
