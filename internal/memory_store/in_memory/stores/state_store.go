package stores

import (
	"sync"
	"time"
)

// stateTTL is the maximum lifetime of a state entry. RFC 6749 §4.1.2
// recommends authorization codes expire within 10 minutes; we apply the
// same TTL to all state entries (codes, nonces, PKCE challenges).
const stateTTL = 10 * time.Minute

type stateEntry struct {
	value     string
	expiresAt time.Time
}

// StateStore stores OAuth state entries with automatic TTL expiration.
type StateStore struct {
	mutex sync.Mutex
	store map[string]stateEntry
}

// NewStateStore create a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		mutex: sync.Mutex{},
		store: make(map[string]stateEntry),
	}
}

// Get returns the value of the key in state store.
// Returns empty string if the key does not exist or has expired.
func (s *StateStore) Get(key string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	entry, ok := s.store[key]
	if !ok {
		return ""
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.store, key)
		return ""
	}
	return entry.value
}

// Set sets the value of the key in state store with a TTL.
func (s *StateStore) Set(key string, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store[key] = stateEntry{
		value:     value,
		expiresAt: time.Now().Add(stateTTL),
	}
}

// Remove removes the key from state store
func (s *StateStore) Remove(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.store, key)
}
