package sessionstore

import (
	"strings"
	"sync"
)

// InMemoryStore is a simple in-memory store for sessions.
type InMemoryStore struct {
	mutex        sync.Mutex
	sessionStore map[string]map[string]string
	stateStore   map[string]string
}

// ClearStore clears the in-memory store.
func (c *InMemoryStore) ClearStore() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessionStore = map[string]map[string]string{}
}

// GetUserSessions returns all the user session token from the in-memory store.
func (c *InMemoryStore) GetUserSessions(userId string) map[string]string {
	// c.mutex.Lock()
	// defer c.mutex.Unlock()
	res := map[string]string{}
	for k, v := range c.stateStore {
		split := strings.Split(v, "@")
		if split[1] == userId {
			res[k] = split[0]
		}
	}

	return res
}

// DeleteAllUserSession deletes all the user sessions from in-memory store.
func (c *InMemoryStore) DeleteAllUserSession(userId string) {
	// c.mutex.Lock()
	// defer c.mutex.Unlock()
	sessions := GetUserSessions(userId)
	for k := range sessions {
		RemoveState(k)
	}
}

// SetState sets the state in the in-memory store.
func (c *InMemoryStore) SetState(key, state string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.stateStore[key] = state
}

// GetState gets the state from the in-memory store.
func (c *InMemoryStore) GetState(key string) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	state := ""
	if stateVal, ok := c.stateStore[key]; ok {
		state = stateVal
	}

	return state
}

// RemoveState removes the state from the in-memory store.
func (c *InMemoryStore) RemoveState(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.stateStore, key)
}
