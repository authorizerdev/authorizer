package stores

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// Maximum entries to keep in session storage
	maxCacheSize = 1000
	// Cache clear interval
	clearInterval = 10 * time.Minute
)

// SessionEntry is the struct for entry stored in store
type SessionEntry struct {
	Value     string
	ExpiresAt int64
}

// SessionStore struct to store the env variables
type SessionStore struct {
	wg    sync.WaitGroup
	mutex sync.Mutex
	store map[string]*SessionEntry
	// stores expireTime: key to remove data when cache is full
	// map is sorted by key so older most entry can be deleted first
	keyIndex map[int64]string
	stop     chan struct{}
}

// NewSessionStore create a new session store
func NewSessionStore() *SessionStore {
	store := &SessionStore{
		mutex:    sync.Mutex{},
		store:    make(map[string]*SessionEntry),
		keyIndex: make(map[int64]string),
		stop:     make(chan struct{}),
	}
	store.wg.Add(1)
	go func() {
		defer store.wg.Done()
		store.clean()
	}()
	return store
}

func (s *SessionStore) clean() {
	t := time.NewTicker(clearInterval)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.mutex.Lock()
			currentTime := time.Now().Unix()
			for k, v := range s.store {
				if v.ExpiresAt < currentTime {
					delete(s.store, k)
					delete(s.keyIndex, v.ExpiresAt)
				}
			}
			s.mutex.Unlock()
		}
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
		// Delete expired items
		delete(s.store, k)
	}
	return ""
}

// Get returns the value of the key in state store
func (s *SessionStore) GetAll(key string) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	currentTime := time.Now().Unix()
	// Match all keys with the given key
	var values []string
	for k, v := range s.store {
		if strings.HasPrefix(k, key) {
			if v.ExpiresAt < currentTime {
				values = append(values, v.Value)
			} else {
				// Delete expired items
				delete(s.store, k)
			}
		}
	}
	return values
}

// Set sets the value of the key in state store
func (s *SessionStore) Set(key string, subKey, value string, expiration int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	k := fmt.Sprintf("%s:%s", key, subKey)
	if _, ok := s.store[k]; !ok {
		// check if there is enough space in cache
		// else delete entries based on FIFO
		if len(s.store) == maxCacheSize {
			// remove older most entry
			sortedKeys := []int64{}
			for ik := range s.keyIndex {
				sortedKeys = append(sortedKeys, ik)
			}
			sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i] < sortedKeys[j] })
			itemToRemove := sortedKeys[0]
			delete(s.store, s.keyIndex[itemToRemove])
			delete(s.keyIndex, itemToRemove)
		}
	}
	s.store[k] = &SessionEntry{
		Value:     value,
		ExpiresAt: expiration,
	}
	s.keyIndex[expiration] = k
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
