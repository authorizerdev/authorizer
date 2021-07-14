package session

import "sync"

type InMemoryStore struct {
	mu    sync.Mutex
	store map[string]string
}

func (c *InMemoryStore) AddToken(userId, token string) {
	c.mu.Lock()
	// delete sessions > 500 // not recommended for production
	if len(c.store) >= 500 {
		c.store = make(map[string]string)
	}
	c.store[userId] = token
	c.mu.Unlock()
}

func (c *InMemoryStore) DeleteToken(userId string) {
	c.mu.Lock()
	delete(c.store, userId)
	c.mu.Unlock()
}

func (c *InMemoryStore) ClearStore() {
	c.mu.Lock()
	c.store = make(map[string]string)
	c.mu.Unlock()
}

func (c *InMemoryStore) GetToken(userId string) string {
	token := ""
	c.mu.Lock()
	if val, ok := c.store[userId]; ok {
		token = val
	}
	c.mu.Unlock()

	return token
}
