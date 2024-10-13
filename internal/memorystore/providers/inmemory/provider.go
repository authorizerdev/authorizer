package inmemory

import (
	"sync"

	"github.com/authorizerdev/authorizer/internal/memorystore/providers/inmemory/stores"
)

type provider struct {
	mutex           sync.Mutex
	sessionStore    *stores.SessionStore
	mfasessionStore *stores.SessionStore
	stateStore      *stores.StateStore
	envStore        *stores.EnvStore
}

// NewInMemoryStore returns a new in-memory store.
func NewInMemoryProvider() (*provider, error) {
	return &provider{
		mutex:           sync.Mutex{},
		envStore:        stores.NewEnvStore(),
		sessionStore:    stores.NewSessionStore(),
		mfasessionStore: stores.NewSessionStore(),
		stateStore:      stores.NewStateStore(),
	}, nil
}
