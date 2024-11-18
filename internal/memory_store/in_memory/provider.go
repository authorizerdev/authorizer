package in_memory

import (
	"sync"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/memory_store/in_memory/stores"
)

// Dependencies struct for in_memory store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	dependencies    Dependencies
	mutex           sync.Mutex
	sessionStore    *stores.SessionStore
	mfasessionStore *stores.SessionStore
	stateStore      *stores.StateStore
	envStore        *stores.EnvStore
}

// NewInMemoryStore returns a new in-memory store.
func NewInMemoryProvider(deps Dependencies) (*provider, error) {
	return &provider{
		dependencies:    deps,
		mutex:           sync.Mutex{},
		envStore:        stores.NewEnvStore(),
		sessionStore:    stores.NewSessionStore(),
		mfasessionStore: stores.NewSessionStore(),
		stateStore:      stores.NewStateStore(),
	}, nil
}
