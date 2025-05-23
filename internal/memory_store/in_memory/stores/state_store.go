package stores

import (
	"sync"
)

// StateStore struct to store the env variables
type StateStore struct {
	mutex sync.Mutex
	store map[string]string
}

// NewStateStore create a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		mutex: sync.Mutex{},
		store: make(map[string]string),
	}
}

// Get returns the value of the key in state store
func (s *StateStore) Get(key string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.store[key]
}

// Set sets the value of the key in state store
func (s *StateStore) Set(key string, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store[key] = value
}

// Remove removes the key from state store
func (s *StateStore) Remove(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.store, key)
}
