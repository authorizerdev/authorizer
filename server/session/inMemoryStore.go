package session

import (
	"log"
	"sync"
)

type InMemoryStore struct {
	mu               sync.Mutex
	store            map[string]map[string]string
	socialLoginState map[string]string
}

func (c *InMemoryStore) AddToken(userId, accessToken, refreshToken string) {
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

	log.Println(c.store)

	c.mu.Unlock()
}

func (c *InMemoryStore) DeleteUserSession(userId string) {
	c.mu.Lock()
	delete(c.store, userId)
	c.mu.Unlock()
}

func (c *InMemoryStore) DeleteToken(userId, accessToken string) {
	c.mu.Lock()
	delete(c.store[userId], accessToken)
	c.mu.Unlock()
}

func (c *InMemoryStore) ClearStore() {
	c.mu.Lock()
	c.store = map[string]map[string]string{}
	c.mu.Unlock()
}

func (c *InMemoryStore) GetToken(userId, accessToken string) string {
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

func (c *InMemoryStore) SetSocialLoginState(key, state string) {
	c.mu.Lock()
	c.socialLoginState[key] = state
	c.mu.Unlock()
}

func (c *InMemoryStore) GetSocialLoginState(key string) string {
	state := ""
	if stateVal, ok := c.socialLoginState[key]; ok {
		state = stateVal
	}

	return state
}

func (c *InMemoryStore) RemoveSocialLoginState(key string) {
	c.mu.Lock()
	delete(c.socialLoginState, key)
	c.mu.Unlock()
}
