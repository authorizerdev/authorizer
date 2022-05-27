package inmemory

import "sync"

type provider struct {
	mutex        sync.Mutex
	sessionStore map[string]map[string]string
	stateStore   map[string]string
}

// NewInMemoryStore returns a new in-memory store.
func NewInMemoryProvider() (*provider, error) {
	return &provider{
		mutex:        sync.Mutex{},
		sessionStore: map[string]map[string]string{},
		stateStore:   map[string]string{},
	}, nil
}
