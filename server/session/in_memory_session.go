package session

import (
	"sync"
)

// InMemoryStore is a simple in-memory store for sessions.
type InMemoryStore struct {
	mu               sync.Mutex
	store            map[string]map[string]string
	socialLoginState map[string]string
}

// AddUserSession adds a user session to the in-memory store.
func (c *InMemoryStore) AddUserSession(userId, accessToken, refreshToken string) {
	c.mu.Lock()
	// delete sessions > 500 // not recommended for production
	if len(c.store) >= 500 {
		c.store = map[string]map[string]string{}
	}
	// check if entry exists in map
	_, exists := c.store[userId]
	if exists {
		tempMap := c.store[userId]
		tempMap[accessToken] = refreshToken
		c.store[userId] = tempMap
	} else {
		tempMap := map[string]string{
			accessToken: refreshToken,
		}
		c.store[userId] = tempMap
	}

	c.mu.Unlock()
}

// DeleteAllUserSession deletes all the user sessions from in-memory store.
func (c *InMemoryStore) DeleteAllUserSession(userId string) {
	c.mu.Lock()
	delete(c.store, userId)
	c.mu.Unlock()
}

// DeleteUserSession deletes the particular user session from in-memory store.
func (c *InMemoryStore) DeleteUserSession(userId, accessToken string) {
	c.mu.Lock()
	delete(c.store[userId], accessToken)
	c.mu.Unlock()
}

// ClearStore clears the in-memory store.
func (c *InMemoryStore) ClearStore() {
	c.mu.Lock()
	c.store = map[string]map[string]string{}
	c.mu.Unlock()
}

// GetUserSession returns the user session token from the in-memory store.
func (c *InMemoryStore) GetUserSession(userId, accessToken string) string {
	token := ""
	c.mu.Lock()
	if sessionMap, ok := c.store[userId]; ok {
		if val, ok := sessionMap[accessToken]; ok {
			token = val
		}
	}
	c.mu.Unlock()

	return token
}

// SetSocialLoginState sets the social login state in the in-memory store.
func (c *InMemoryStore) SetSocialLoginState(key, state string) {
	c.mu.Lock()
	c.socialLoginState[key] = state
	c.mu.Unlock()
}

// GetSocialLoginState gets the social login state from the in-memory store.
func (c *InMemoryStore) GetSocialLoginState(key string) string {
	state := ""
	if stateVal, ok := c.socialLoginState[key]; ok {
		state = stateVal
	}

	return state
}

// RemoveSocialLoginState removes the social login state from the in-memory store.
func (c *InMemoryStore) RemoveSocialLoginState(key string) {
	c.mu.Lock()
	delete(c.socialLoginState, key)
	c.mu.Unlock()
}
