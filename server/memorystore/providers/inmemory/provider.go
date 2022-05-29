package inmemory

import (
	"sync"
)

type provider struct {
	mutex        sync.Mutex
	sessionStore map[string]map[string]string
	stateStore   map[string]string
	envStore     *EnvStore
}

// NewInMemoryStore returns a new in-memory store.
func NewInMemoryProvider() (*provider, error) {
	return &provider{
		mutex:        sync.Mutex{},
		sessionStore: map[string]map[string]string{},
		stateStore:   map[string]string{},
		envStore: &EnvStore{
			mutex: sync.Mutex{},
			store: map[string]interface{}{},
		},
	}, nil
}
