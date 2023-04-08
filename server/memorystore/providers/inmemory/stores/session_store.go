package stores

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// Maximum entries to keep in session storage
	maxCacheSize = 1000
)

// SessionEntry is the struct for entry stored in store
type SessionEntry struct {
	Value     string
	ExpiresAt int64
}

// SessionStore struct to store the env variables
type SessionStore struct {
	mutex        sync.Mutex
	store        map[string]*SessionEntry
	itemsToEvict []string
}

// NewSessionStore create a new session store
func NewSessionStore() *SessionStore {
	return &SessionStore{
		mutex: sync.Mutex{},
		store: make(map[string]*SessionEntry),
	}
}

// Get returns the value of the key in state store
func (s *SessionStore) Get(key, subKey string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	currentTime := time.Now().Unix()
	k := fmt.Sprintf("%s:%s", key, subKey)
	if v, ok := s.store[k]; ok {
		if v.ExpiresAt > currentTime {
			return v.Value
		}
		s.itemsToEvict = append(s.itemsToEvict, k)
	}
	return ""
}

// Set sets the value of the key in state store
func (s *SessionStore) Set(key string, subKey, value string, expiration int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	k := fmt.Sprintf("%s:%s", key, subKey)
	if _, ok := s.store[k]; !ok {
		s.store[k] = &SessionEntry{
			Value:     value,
			ExpiresAt: expiration,
			// TODO add expire time
		}
	}
	s.store[k] = &SessionEntry{
		Value:     value,
		ExpiresAt: expiration,
		// TODO add expire time
	}
}

// RemoveAll all values for given key
func (s *SessionStore) RemoveAll(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for k := range s.store {
		if strings.Contains(k, key) {
			delete(s.store, k)
		}
	}
}

// Remove value for given key and subkey
func (s *SessionStore) Remove(key, subKey string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	k := fmt.Sprintf("%s:%s", key, subKey)
	delete(s.store, k)
}

// RemoveByNamespace to delete session for a given namespace example google,github
func (s *SessionStore) RemoveByNamespace(namespace string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for key := range s.store {
		if strings.Contains(key, namespace+":") {
			delete(s.store, key)
		}
	}
	return nil
}
