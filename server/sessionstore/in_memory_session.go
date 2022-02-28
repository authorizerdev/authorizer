package sessionstore

import (
	"sync"
)

// InMemoryStore is a simple in-memory store for sessions.
type InMemoryStore struct {
	mutex        sync.Mutex
	sessionStore map[string]map[string]string
	stateStore   map[string]string
}

// AddUserSession adds a user session to the in-memory store.
func (c *InMemoryStore) AddUserSession(userId, accessToken, refreshToken string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// delete sessions > 500 // not recommended for production
	if len(c.sessionStore) >= 500 {
		c.sessionStore = map[string]map[string]string{}
	}
	// check if entry exists in map
	_, exists := c.sessionStore[userId]
	if exists {
		tempMap := c.sessionStore[userId]
		tempMap[accessToken] = refreshToken
		c.sessionStore[userId] = tempMap
	} else {
		tempMap := map[string]string{
			accessToken: refreshToken,
		}
		c.sessionStore[userId] = tempMap
	}
}

// DeleteAllUserSession deletes all the user sessions from in-memory store.
func (c *InMemoryStore) DeleteAllUserSession(userId string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.sessionStore, userId)
}

// DeleteUserSession deletes the particular user session from in-memory store.
func (c *InMemoryStore) DeleteUserSession(userId, accessToken string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.sessionStore[userId], accessToken)
}

// ClearStore clears the in-memory store.
func (c *InMemoryStore) ClearStore() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessionStore = map[string]map[string]string{}
}

// GetUserSession returns the user session token from the in-memory store.
func (c *InMemoryStore) GetUserSession(userId, accessToken string) string {
	// c.mutex.Lock()
	// defer c.mutex.Unlock()

	token := ""
	if sessionMap, ok := c.sessionStore[userId]; ok {
		if val, ok := sessionMap[accessToken]; ok {
			token = val
		}
	}

	return token
}

// GetUserSessions returns all the user session token from the in-memory store.
func (c *InMemoryStore) GetUserSessions(userId string) map[string]string {
	// c.mutex.Lock()
	// defer c.mutex.Unlock()

	sessionMap, ok := c.sessionStore[userId]
	if !ok {
		return nil
	}

	return sessionMap
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
