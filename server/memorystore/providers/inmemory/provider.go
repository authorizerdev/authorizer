package inmemory

import (
	"sync"

	"github.com/authorizerdev/authorizer/server/memorystore/providers/inmemory/stores"
)

type provider struct {
	mutex        sync.Mutex
	sessionStore *stores.SessionStore
	stateStore   *stores.StateStore
	envStore     *stores.EnvStore
}

// NewInMemoryStore returns a new in-memory store.
func NewInMemoryProvider() (*provider, error) {
	return &provider{
		mutex:        sync.Mutex{},
		envStore:     stores.NewEnvStore(),
		sessionStore: stores.NewSessionStore(),
		stateStore:   stores.NewStateStore(),
	}, nil
}
