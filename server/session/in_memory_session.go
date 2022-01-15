package session

import (
	"sync"
)

// InMemoryStore is a simple in-memory store for sessions.
type InMemoryStore struct {
	mutex            sync.Mutex
	store            map[string]map[string]string
	socialLoginState map[string]string
}

// AddUserSession adds a user session to the in-memory store.
func (c *InMemoryStore) AddUserSession(userId, accessToken, refreshToken string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
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
}

// DeleteAllUserSession deletes all the user sessions from in-memory store.
func (c *InMemoryStore) DeleteAllUserSession(userId string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.store, userId)
}

// DeleteUserSession deletes the particular user session from in-memory store.
func (c *InMemoryStore) DeleteUserSession(userId, accessToken string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.store[userId], accessToken)
}

// ClearStore clears the in-memory store.
func (c *InMemoryStore) ClearStore() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.store = map[string]map[string]string{}
}

// GetUserSession returns the user session token from the in-memory store.
func (c *InMemoryStore) GetUserSession(userId, accessToken string) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	token := ""
	if sessionMap, ok := c.store[userId]; ok {
		if val, ok := sessionMap[accessToken]; ok {
			token = val
		}
	}

	return token
}

// SetSocialLoginState sets the social login state in the in-memory store.
func (c *InMemoryStore) SetSocialLoginState(key, state string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.socialLoginState[key] = state
}

// GetSocialLoginState gets the social login state from the in-memory store.
func (c *InMemoryStore) GetSocialLoginState(key string) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	state := ""
	if stateVal, ok := c.socialLoginState[key]; ok {
		state = stateVal
	}

	return state
}

// RemoveSocialLoginState removes the social login state from the in-memory store.
func (c *InMemoryStore) RemoveSocialLoginState(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.socialLoginState, key)
}
