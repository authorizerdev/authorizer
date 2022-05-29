package inmemory

import "sync"

// EnvStore struct to store the env variables
type EnvStore struct {
	mutex sync.Mutex
	store map[string]interface{}
}

// UpdateEnvStore to update the whole env store object
func (e *EnvStore) UpdateStore(store map[string]interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// just override the keys + new keys

	for key, value := range store {
		e.store[key] = value
	}
}

// GetStore returns the env store
func (e *EnvStore) GetStore() map[string]interface{} {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.store
}

// Get returns the value of the key in evn store
func (s *EnvStore) Get(key string) interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.store[key]
}

// Set sets the value of the key in env store
func (s *EnvStore) Set(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.store[key] = value
}
